package commoninterpreters

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

const (
	ID      = "common-interpreters"
	Name    = "Common Interpreters"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	r.RegisterSource(sdk.SourceRef{
		Name: "Vortex common-interpreters extension source",
		URL:  "https://github.com/Nexus-Mods/Vortex/tree/2349a17900a37c2120e90733045dc6b303135b89/extensions/common-interpreters/src",
	})
	message := "Mirrors Vortex common-interpreters by resolving script/tool launch commands through extension-owned interpreter registrations."
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "jar",
		Name:           "Java archive",
		FileExtensions: []string{".jar"},
		Command:        "java",
		Arguments:      []string{"-jar", "{path}"},
		Platforms:      []string{"linux", "windows"},
		Status:         sdk.CapabilityStatusReady,
		Message:        message,
		Resolver:       resolveJava,
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "python",
		Name:           "Python script",
		FileExtensions: []string{".py"},
		Command:        "python",
		Arguments:      []string{"{path}"},
		Platforms:      []string{"linux", "windows"},
		Status:         sdk.CapabilityStatusReady,
		Message:        message,
		Resolver:       resolvePython,
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "vbs",
		Name:           "Windows Script Host VBScript",
		FileExtensions: []string{".vbs"},
		Command:        "cscript.exe",
		Arguments:      []string{"{path}"},
		Platforms:      []string{"windows"},
		Status:         sdk.CapabilityStatusReady,
		Message:        message,
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "cmd",
		Name:           "Windows command script",
		FileExtensions: []string{".cmd"},
		Command:        "cmd.exe",
		Arguments:      []string{"/K", "{path}"},
		Platforms:      []string{"windows"},
		Status:         sdk.CapabilityStatusReady,
		Message:        message,
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "bat",
		Name:           "Windows batch script",
		FileExtensions: []string{".bat"},
		Command:        "cmd.exe",
		Arguments:      []string{"/K", "{path}"},
		Platforms:      []string{"windows"},
		Status:         sdk.CapabilityStatusReady,
		Message:        message,
	})
}

func resolveJava(input sdk.InterpreterInput) (sdk.InterpreterResult, error) {
	javaHome := strings.TrimSpace(os.Getenv("JAVA_HOME"))
	if javaHome != "" {
		candidate := filepath.Join(javaHome, "bin", executableName("java", input.Platform))
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return sdk.InterpreterResult{Command: candidate, Arguments: []string{"-jar", input.ExecutablePath}}, nil
		}
	}
	if path, err := exec.LookPath(executableName("java", input.Platform)); err == nil {
		return sdk.InterpreterResult{Command: path, Arguments: []string{"-jar", input.ExecutablePath}}, nil
	}
	return sdk.InterpreterResult{}, errors.New("java interpreter is not installed")
}

func resolvePython(input sdk.InterpreterInput) (sdk.InterpreterResult, error) {
	if path, err := exec.LookPath(executableName("python", input.Platform)); err == nil {
		return sdk.InterpreterResult{Command: path, Arguments: []string{input.ExecutablePath}}, nil
	}
	return sdk.InterpreterResult{}, errors.New("python interpreter is not installed")
}

func executableName(name, platform string) string {
	if strings.EqualFold(strings.TrimSpace(platform), "windows") {
		return name + ".exe"
	}
	return name
}
