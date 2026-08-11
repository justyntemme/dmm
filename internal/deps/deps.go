package deps

import "os/exec"

type Tool struct {
	Name           string `json:"name"`
	Command        string `json:"command"`
	Installed      bool   `json:"installed"`
	Path           string `json:"path,omitempty"`
	Description    string `json:"description"`
	PackageName    string `json:"package_name,omitempty"`
	InstallCommand string `json:"install_command,omitempty"`
	InstallHint    string `json:"install_hint,omitempty"`
	DocsURL        string `json:"docs_url,omitempty"`
}

func CheckArchiveTools() []Tool {
	tools := []Tool{
		{
			Name:           "7-Zip",
			Command:        "7z",
			Description:    "Extracts .7z and many Nexus archive formats.",
			PackageName:    "p7zip",
			InstallCommand: "sudo pacman -S --needed p7zip",
			InstallHint:    "Install p7zip through a safe package source for your SteamOS setup.",
			DocsURL:        "https://wiki.archlinux.org/title/P7zip",
		},
		{
			Name:           "bsdtar",
			Command:        "bsdtar",
			Description:    "Extracts tar and zip archives.",
			PackageName:    "libarchive",
			InstallCommand: "sudo pacman -S --needed libarchive",
			InstallHint:    "Install libarchive/bsdtar through a safe package source for your SteamOS setup.",
			DocsURL:        "https://man.archlinux.org/man/bsdtar.1",
		},
		{
			Name:           "unzip",
			Command:        "unzip",
			Description:    "Extracts .zip archives.",
			PackageName:    "unzip",
			InstallCommand: "sudo pacman -S --needed unzip",
			InstallHint:    "Install unzip through a safe package source for your SteamOS setup.",
			DocsURL:        "https://man.archlinux.org/man/unzip.1.en",
		},
		{
			Name:           "unrar",
			Command:        "unrar",
			Description:    "Extracts .rar archives when available.",
			PackageName:    "unrar",
			InstallCommand: "sudo pacman -S --needed unrar",
			InstallHint:    "Install unrar through a safe package source for your SteamOS setup.",
			DocsURL:        "https://man.archlinux.org/man/unrar.1.en",
		},
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
