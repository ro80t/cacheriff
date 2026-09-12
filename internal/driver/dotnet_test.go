package driver

import (
	"encoding/json"
	"testing"
)

func TestParseDotnetNugetLocalsLines(t *testing.T) {
	// Real `dotnet nuget locals all --list` output.
	out := `http-cache: C:\Users\me\AppData\Local\NuGet\v3-cache
global-packages: C:\Users\me\.nuget\packages\
temp: C:\Users\me\AppData\Local\Temp\NuGetScratch
plugins-cache: C:\Users\me\AppData\Local\NuGet\plugins-cache
`
	locals := parseDotnetNugetLocals(out)
	want := map[string]string{
		"http-cache":      `C:\Users\me\AppData\Local\NuGet\v3-cache`,
		"global-packages": `C:\Users\me\.nuget\packages\`,
		"temp":            `C:\Users\me\AppData\Local\Temp\NuGetScratch`,
		"plugins-cache":   `C:\Users\me\AppData\Local\NuGet\plugins-cache`,
	}
	if len(locals) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(locals), len(want), locals)
	}
	for k, v := range want {
		if locals[k] != v {
			t.Errorf("%s: got %q, want %q", k, locals[k], v)
		}
	}
}

func TestDotnetToolListJSON(t *testing.T) {
	// Real `dotnet tool list -g --format json` output.
	out := []byte(`{"version":1,"data":[{"packageId":"dotnet-ef","version":"9.0.5","commands":["dotnet-ef"]}]}`)
	var parsed dotnetToolListOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.Data) != 1 || parsed.Data[0].PackageID != "dotnet-ef" || parsed.Data[0].Version != "9.0.5" {
		t.Errorf("got %+v, want one dotnet-ef 9.0.5 entry", parsed.Data)
	}
}

func TestDotnetToolListJSONEmpty(t *testing.T) {
	var parsed dotnetToolListOutput
	if err := json.Unmarshal([]byte(`{"version":1,"data":[]}`), &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(parsed.Data) != 0 {
		t.Errorf("got %+v, want no entries", parsed.Data)
	}
}

func TestDotnetProjectAssetsJSON(t *testing.T) {
	// Real (trimmed) obj/project.assets.json produced by `dotnet add
	// package Newtonsoft.Json`.
	data := []byte(`{
		"version": 3,
		"libraries": {
			"Newtonsoft.Json/13.0.3": {
				"sha512": "...",
				"type": "package",
				"path": "newtonsoft.json/13.0.3",
				"files": []
			}
		}
	}`)
	var assets dotnetProjectAssets
	if err := json.Unmarshal(data, &assets); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	lib, ok := assets.Libraries["Newtonsoft.Json/13.0.3"]
	if !ok {
		t.Fatalf("missing Newtonsoft.Json/13.0.3 in %+v", assets.Libraries)
	}
	if lib.Type != "package" || lib.Path != "newtonsoft.json/13.0.3" {
		t.Errorf("got %+v, want type=package path=newtonsoft.json/13.0.3", lib)
	}
}
