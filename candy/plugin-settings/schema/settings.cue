// plugin-settings's OWN self-contained CUE schema — the SINGLE SOURCE for this plugin's
// declaration surface, served over the Describe channel (there is no schema-less
// plugin). SELF-CONTAINED: it references no base def, so it compiles STANDALONE (the
// property the SDK's serve-side compile and `cue exp gengotypes` both need).
//
// `command:settings`'s authored input is its pass-through CLI grammar (the
// get/set/list/reset/path subcommands), not a structured plugin_input, so this schema
// DOCUMENTS the command contract and the runtime-config surface the plugin owns.
#SettingsPlugin: {
	// The command word the plugin serves.
	command: "settings"

	// What the command does, in one line (the public-docs surface).
	contract: string & !=""

	// The subcommands the plugin's grammar owns.
	subcommands: ["get", "set", "list", "reset", "path"]
}
