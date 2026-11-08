// Command d2sign makes and checks the signed release manifest of the
// self-update design (docs/self-update-release.md, docs/research/self-update-design.md
// 3.2, 3.3). Plain Ed25519, standard library only. It never prints a private key.
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"d2lod-bot/internal/selfupdate"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

const usage = `usage: d2sign <command> [flags]
  keygen   -dir D [-name N]                 write N.key (0600) and N.pub, print the public key and fingerprint
  manifest -dir D -version V -seq N ...     build an unsigned MANIFEST.json from the asset files in D
  revoked  -seq N [-revoke V]... [-min-version V] -out F   build a REVOKED document
  sign     (-key F | -key-env NAME) -in F [-out F]         write the signature (default F.sig)
  verify   -pub F[,F] -in F [-sig F] [-dir D]               check a signature; with a manifest and -dir also its hashes
  summary  -in F                                            print the review summary of a manifest or REVOKED file`

func run(args []string, out, errw io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "keygen":
		err = cmdKeygen(args[1:], out)
	case "manifest":
		err = cmdManifest(args[1:], out)
	case "revoked":
		err = cmdRevoked(args[1:], out)
	case "sign":
		err = cmdSign(args[1:], out)
	case "verify":
		err = cmdVerify(args[1:], out)
	case "summary":
		err = cmdSummary(args[1:], out)
	case "-h", "--help", "help":
		fmt.Fprintln(out, usage)
		return 0
	default:
		fmt.Fprintf(errw, "d2sign: unknown command %q\n%s\n", args[0], usage)
		return 2
	}
	if err != nil {
		fmt.Fprintln(errw, "d2sign:", err)
		return 1
	}
	return 0
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

type multi []string

func (m *multi) String() string     { return strings.Join(*m, ",") }
func (m *multi) Set(s string) error { *m = append(*m, s); return nil }

func cmdKeygen(args []string, out io.Writer) error {
	fs := newFlags("keygen")
	dir := fs.String("dir", "", "directory for the key files")
	name := fs.String("name", "d2sign", "file name stem")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errors.New("keygen: -dir is required")
	}
	if err := os.MkdirAll(*dir, 0o700); err != nil {
		return err
	}
	pub, priv, err := selfupdate.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	// O_EXCL: never overwrite an existing key.
	if err := writeNew(filepath.Join(*dir, *name+".key"), selfupdate.EncodePrivateKey(priv), 0o600); err != nil {
		return err
	}
	if err := writeNew(filepath.Join(*dir, *name+".pub"), selfupdate.EncodePublicKey(pub), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "public key file: %s\nfingerprint: %s\n", filepath.Join(*dir, *name+".pub"), selfupdate.Fingerprint(pub))
	fmt.Fprintf(out, "private key file: %s (mode 0600, not printed; store it as the environment secret, then delete or encrypt the copy)\n", filepath.Join(*dir, *name+".key"))
	return nil
}

func writeNew(p string, b []byte, mode os.FileMode) error {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func cmdManifest(args []string, out io.Writer) error {
	fs := newFlags("manifest")
	dir := fs.String("dir", "", "directory holding the asset files")
	m := selfupdate.Manifest{Schema: selfupdate.SchemaVersion}
	fs.StringVar(&m.Version, "version", "", "release version x.y.z")
	fs.Int64Var(&m.Seq, "seq", 0, "monotonic release counter")
	fs.StringVar(&m.Issued, "issued", "", "issue time, RFC 3339 (explicit, so the bytes are reproducible)")
	fs.StringVar(&m.Channel, "channel", "stable", "channel")
	fs.StringVar(&m.Notes, "notes", "", "short plain-text changelog")
	fs.IntVar(&m.MinImage, "min-image", 0, "minimum image ABI level")
	fs.IntVar(&m.Compat.BotAPI, "bot-api", 0, "compat.bot_api")
	fs.IntVar(&m.Compat.D2DCtl, "d2d-ctl", 0, "compat.d2d_ctl")
	fs.IntVar(&m.Compat.DataSchema, "data-schema", 0, "compat.data_schema")
	fs.StringVar(&m.Source.Bot, "src-bot", "", "bot source tag/commit")
	fs.StringVar(&m.Source.Fork, "src-d2d", "", "d2d fork tag/commit")
	fs.StringVar(&m.Source.Offer, "offer", "SOURCE.txt", "source offer file name")
	fs.StringVar(&m.Revoke.MinVersion, "min-version", "", "refuse every version below this")
	var files, revoke multi
	fs.Var(&files, "file", "asset as name=dest (repeat)")
	fs.Var(&revoke, "revoke", "revoked version (repeat)")
	outp := fs.String("out", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *dir == "" {
		return errors.New("manifest: -dir is required")
	}
	for _, spec := range files {
		name, dest, ok := strings.Cut(spec, "=")
		if !ok {
			return fmt.Errorf("manifest: -file %q: want name=dest", spec)
		}
		f := selfupdate.File{Name: name, Dest: dest}
		fh, err := os.Open(filepath.Join(*dir, name))
		if err != nil {
			return err
		}
		f.SHA256, f.Size, err = selfupdate.HashReader(fh)
		fh.Close()
		if err != nil {
			return err
		}
		m.Files = append(m.Files, f)
	}
	m.Revoke.Versions = revoke
	b, err := m.Marshal()
	if err != nil {
		return err
	}
	return emit(*outp, b, out)
}

func cmdRevoked(args []string, out io.Writer) error {
	fs := newFlags("revoked")
	r := selfupdate.Revoked{Schema: selfupdate.SchemaVersion}
	fs.Int64Var(&r.Seq, "seq", 0, "monotonic counter")
	fs.StringVar(&r.MinVersion, "min-version", "", "refuse every version below this")
	var revoke multi
	fs.Var(&revoke, "revoke", "revoked version (repeat)")
	outp := fs.String("out", "", "output file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if r.Seq < 1 {
		return errors.New("revoked: -seq >= 1 is required")
	}
	r.Versions = revoke
	b, err := r.Marshal()
	if err != nil {
		return err
	}
	return emit(*outp, b, out)
}

func emit(path string, b []byte, out io.Writer) error {
	if path == "" {
		_, err := out.Write(b)
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func cmdSign(args []string, out io.Writer) error {
	fs := newFlags("sign")
	keyFile := fs.String("key", "", "private key file (must be 0600)")
	keyEnv := fs.String("key-env", "", "environment variable holding the private key file body")
	in := fs.String("in", "", "document to sign")
	outp := fs.String("out", "", "signature file (default: <in>.sig)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *in == "" || (*keyFile == "") == (*keyEnv == "") {
		return errors.New("sign: -in and exactly one of -key, -key-env are required")
	}
	var raw []byte
	if *keyEnv != "" {
		v := os.Getenv(*keyEnv)
		if v == "" {
			return fmt.Errorf("sign: environment variable %s is empty", *keyEnv)
		}
		raw = []byte(v)
	} else {
		st, err := os.Stat(*keyFile)
		if err != nil {
			return err
		}
		if st.Mode().Perm()&0o077 != 0 {
			return fmt.Errorf("sign: %s is readable by group or others; chmod 600", *keyFile)
		}
		if raw, err = os.ReadFile(*keyFile); err != nil {
			return err
		}
	}
	priv, err := selfupdate.DecodePrivateKey(raw)
	if err != nil {
		return err
	}
	doc, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	if err := checkDocument(doc); err != nil {
		return err
	}
	p := *outp
	if p == "" {
		p = *in + ".sig"
	}
	if err := os.WriteFile(p, selfupdate.Sign(priv, doc), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "signed %s with key %s -> %s\n", *in, selfupdate.Fingerprint(priv.Public().(ed25519.PublicKey)), p)
	return nil
}

// checkDocument refuses to sign anything that is not a valid manifest or
// REVOKED document, so a stray file can never get a release signature.
func checkDocument(doc []byte) error {
	if _, err := selfupdate.ParseManifest(doc); err == nil {
		return nil
	}
	if _, err := selfupdate.ParseRevoked(doc); err == nil {
		return nil
	}
	return errors.New("sign: input is neither a valid manifest nor a REVOKED document")
}

func cmdVerify(args []string, out io.Writer) error {
	fs := newFlags("verify")
	pubs := fs.String("pub", "", "public key file(s), comma separated")
	in := fs.String("in", "", "signed document")
	sig := fs.String("sig", "", "signature file (default: <in>.sig)")
	dir := fs.String("dir", "", "for a manifest: directory whose asset hashes are checked")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *pubs == "" || *in == "" {
		return errors.New("verify: -pub and -in are required")
	}
	var keys []ed25519.PublicKey
	for _, p := range strings.Split(*pubs, ",") {
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		k, err := selfupdate.DecodePublicKey(b)
		if err != nil {
			return err
		}
		keys = append(keys, k)
	}
	doc, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	sp := *sig
	if sp == "" {
		sp = *in + ".sig"
	}
	sb, err := os.ReadFile(sp)
	if err != nil {
		return err
	}
	if err := selfupdate.Verify(keys, doc, sb); err != nil {
		return err
	}
	if m, err := selfupdate.ParseManifest(doc); err == nil && *dir != "" {
		if err := m.VerifyFiles(*dir); err != nil {
			return err
		}
		fmt.Fprintf(out, "ok: signature and %d file hashes\n", len(m.Files))
		return nil
	}
	fmt.Fprintln(out, "ok: signature")
	return nil
}

func cmdSummary(args []string, out io.Writer) error {
	fs := newFlags("summary")
	in := fs.String("in", "", "manifest or REVOKED file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	doc, err := os.ReadFile(*in)
	if err != nil {
		return err
	}
	if m, err := selfupdate.ParseManifest(doc); err == nil {
		fmt.Fprintf(out, "release %s (seq %d, channel %s, issued %s)\n", m.Version, m.Seq, m.Channel, m.Issued)
		fmt.Fprintf(out, "min image %d; compat bot_api %d d2d_ctl %d data_schema %d\n", m.MinImage, m.Compat.BotAPI, m.Compat.D2DCtl, m.Compat.DataSchema)
		fmt.Fprintf(out, "source bot %s, d2d %s, offer %s\n", m.Source.Bot, m.Source.Fork, m.Source.Offer)
		for _, f := range m.Files {
			fmt.Fprintf(out, "file %s -> %s  %d bytes  sha256 %s\n", f.Name, f.Dest, f.Size, f.SHA256)
		}
		if len(m.Revoke.Versions) > 0 || m.Revoke.MinVersion != "" {
			fmt.Fprintf(out, "revokes %v, min version %q\n", m.Revoke.Versions, m.Revoke.MinVersion)
		}
		fmt.Fprintf(out, "notes: %s\n", m.Notes)
		return nil
	}
	if r, err := selfupdate.ParseRevoked(doc); err == nil {
		fmt.Fprintf(out, "REVOKED (seq %d): versions %v, min version %q\n", r.Seq, r.Versions, r.MinVersion)
		return nil
	}
	return errors.New("summary: not a manifest or REVOKED document")
}
