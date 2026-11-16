package lima

const GuestScript = `export PATH="$HOME/.local/bin:$PATH"
if [[ -f "$HOME/.config/vbx/environment.sh" ]]; then
 source "$HOME/.config/vbx/environment.sh"
fi
exec "$@"`

func ShellArgv(instance, workdir string, argv []string) []string {
	return append([]string{"shell", "--workdir=" + workdir, instance, "--", "bash", "-eu", "-c", GuestScript, "vbx-exec"}, argv...)
}
func SSHHost(instance string) string { return "lima-" + instance }
