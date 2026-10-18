package auth

import (
	"os/exec"
	"runtime"
)

// OpenBrowser は既定のブラウザでURLを開く。失敗してもURLは表示済みなので致命的ではない。
// テストで差し替えられるよう変数にしている。
var OpenBrowser = openBrowser

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
