# d2idle on the NAS

`my-d2idle.xml` is the Docker-tab template for the d2idle container (bridge network, only port 8431, `/data` rw, `/src` ro,
4 GB / 3 CPUs, cap-drop ALL, stop timeout 90 s, autostart off). Full steps and measurements: `docs/nas-container.md`.

1. Load the image (built on a build host): `ssh worker.example docker save d2idle:dev | ssh <nas-user>@nas.example docker load`
2. Stage your own game install (never in the image), without `default.key`:
   `tar cf - --exclude=default.key --exclude='*.log' . | ssh <nas-user>@nas.example 'tar xf - -C /mnt/user/appdata/d2idle/game'`
3. `mkdir -p /mnt/user/appdata/d2idle/data && chown 1000:1000 /mnt/user/appdata/d2idle/data`
4. Copy the template to `/boot/config/plugins/dockerMan/templates-user/my-d2idle.xml`, then
   `bash <appdata>/tpl-apply.sh d2idle` (the NAS-side template helper) (this also starts it; `--dry-run` prints the docker command).
5. Add `d2idle` to a Folder View folder (`/boot/config/plugins/folder.view/docker.json`, "Games").
