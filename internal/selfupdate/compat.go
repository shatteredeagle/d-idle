package selfupdate

// BuildCompat is what this build of the bot and the boot shim speak: the compatibility pairs a release may not
// exceed and the data schema it must match (Gate). The first release sets the real numbers; they change together
// with the code that needs them (UNVERIFIED against an image until the first release).
var BuildCompat = Compat{BotAPI: 1, D2DCtl: 1, DataSchema: 1}
