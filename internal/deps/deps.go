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
			Name:        "DMM LOOT sorter",
			Command:     "dmm-loot-sorter",
			Description: "Sorts Bethesda/Gamebryo plugin load order through the bundled libloot helper.",
			InstallHint: "This helper is bundled with DMM release packages. Reinstall the Decky package if it is missing.",
			DocsURL:     "https://loot.github.io/",
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
