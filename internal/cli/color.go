package cli

import "os"

var color = getColorEnabled()

func isTTY() bool {
	fileInfo, _ := os.Stdout.Stat()
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// See <https://bixense.com/clicolors/>.
func getColorEnabled() bool {
	force, _ := os.LookupEnv("CLICOLOR_FORCE")
	if force != "" {
		return force != "0"
	}
	prefer, _ := os.LookupEnv("CLICOLOR")
	return isTTY() && prefer != "0"
}
