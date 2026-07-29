package deps

import "os/exec"

type Tool struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Installed   bool   `json:"installed"`
	Path        string `json:"path,omitempty"`
	Description string `json:"description"`
	InstallHint string `json:"install_hint,omitempty"`
}

func CheckArchiveTools() []Tool {
	tools := []Tool{
		{Name: "7-Zip", Command: "7z", Description: "Extracts .7z and many Nexus archive formats.", InstallHint: "Install p7zip/7zip through a safe package source for your SteamOS setup."},
		{Name: "bsdtar", Command: "bsdtar", Description: "Extracts tar/zip archives and is useful as a fallback.", InstallHint: "Install libarchive/bsdtar through a safe package source for your SteamOS setup."},
		{Name: "unzip", Command: "unzip", Description: "Extracts .zip archives.", InstallHint: "Install unzip through a safe package source for your SteamOS setup."},
		{Name: "unrar", Command: "unrar", Description: "Extracts .rar archives when available.", InstallHint: "Install unrar through a safe package source for your SteamOS setup."},
	}
	for i := range tools {
		path, err := exec.LookPath(tools[i].Command)
		if err == nil {
			tools[i].Installed = true
			tools[i].Path = path
		}
	}
	return tools
}
