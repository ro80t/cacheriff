package driver

import (
	"encoding/xml"
	"testing"
)

func TestMavenSettingsLocalRepository(t *testing.T) {
	// Real shape of ~/.m2/settings.xml's override.
	data := []byte(`<settings xmlns="http://maven.apache.org/SETTINGS/1.0.0">
  <localRepository>D:\maven-repo</localRepository>
</settings>`)
	var s mavenSettings
	if err := xml.Unmarshal(data, &s); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	if s.LocalRepository != `D:\maven-repo` {
		t.Errorf("got %q, want D:\\maven-repo", s.LocalRepository)
	}
}

func TestMavenSettingsNoLocalRepository(t *testing.T) {
	data := []byte(`<settings><servers/></settings>`)
	var s mavenSettings
	if err := xml.Unmarshal(data, &s); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}
	if s.LocalRepository != "" {
		t.Errorf("got %q, want empty", s.LocalRepository)
	}
}
