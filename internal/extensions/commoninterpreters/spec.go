package commoninterpreters

import "github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"

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
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "jar",
		Name:           "Java archive",
		FileExtensions: []string{".jar"},
		Command:        "java",
		Arguments:      []string{"-jar", "{path}"},
		Platforms:      []string{"linux", "windows"},
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "python",
		Name:           "Python script",
		FileExtensions: []string{".py"},
		Command:        "python",
		Arguments:      []string{"{path}"},
		Platforms:      []string{"linux", "windows"},
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "vbs",
		Name:           "Windows Script Host VBScript",
		FileExtensions: []string{".vbs"},
		Command:        "cscript.exe",
		Arguments:      []string{"{path}"},
		Platforms:      []string{"windows"},
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "cmd",
		Name:           "Windows command script",
		FileExtensions: []string{".cmd"},
		Command:        "cmd.exe",
		Arguments:      []string{"/K", "{path}"},
		Platforms:      []string{"windows"},
	})
	r.RegisterInterpreter(sdk.InterpreterSpec{
		ID:             "bat",
		Name:           "Windows batch script",
		FileExtensions: []string{".bat"},
		Command:        "cmd.exe",
		Arguments:      []string{"/K", "{path}"},
		Platforms:      []string{"windows"},
	})
}
