package config

import "testing"

func TestParseTarget(t *testing.T) {
	ok := map[string]Connection{
		"nas.lan":                 {Host: "nas.lan", Port: 21, Protocol: "FTP"},
		"ana@nas.lan":             {Host: "nas.lan", User: "ana", Port: 21, Protocol: "FTP"},
		"ana@nas.lan:22":          {Host: "nas.lan", User: "ana", Port: 22, Protocol: "SFTP"},
		"ana@nas.lan:990":         {Host: "nas.lan", User: "ana", Port: 990, Protocol: "FTPS"},
		"nas.lan:2121":            {Host: "nas.lan", Port: 2121, Protocol: "FTP"},
		"sftp://nas.lan":          {Host: "nas.lan", Port: 22, Protocol: "SFTP"},
		"SFTP://ana@nas.lan:2222": {Host: "nas.lan", User: "ana", Port: 2222, Protocol: "SFTP"},
		"ftp://ana@nas.lan:22":    {Host: "nas.lan", User: "ana", Port: 22, Protocol: "FTP"},
		"ftps://nas.lan":          {Host: "nas.lan", Port: 21, Protocol: "FTPS"},
		"ana@[2001:db8::1]:22":    {Host: "2001:db8::1", User: "ana", Port: 22, Protocol: "SFTP"},
		"ana@nas.lan/":            {Host: "nas.lan", User: "ana", Port: 21, Protocol: "FTP"},
	}
	for arg, want := range ok {
		got, err := ParseTarget(arg)
		if err != nil || got != want {
			t.Errorf("%q = %+v, %v; want %+v", arg, got, err, want)
		}
	}

	for _, arg := range []string{"", "ana:secret@nas.lan", "sftp://ana:secret@nas.lan", "nas.lan:0", "nas.lan:99999",
		"nas.lan:http", "http://nas.lan", "sftp://nas.lan/home/ana", "@nas.lan", "sftp://"} {
		if got, err := ParseTarget(arg); err == nil {
			t.Errorf("%q accepted as %+v", arg, got)
		}
	}
}

func TestResolvePrefersSavedNames(t *testing.T) {
	fav := Connection{Name: "nas.lan", Host: "10.0.0.5", Port: 22, Protocol: "SFTP"}
	ssh := Connection{Name: "web", Host: "web.example.com", Port: 22, Protocol: "SFTP"}

	if got, _ := Resolve("nas.lan", []Connection{fav}, []Connection{ssh}); got != fav {
		t.Errorf("a favorite named like a host lost to the host: %+v", got)
	}
	if got, _ := Resolve("web", []Connection{fav}, []Connection{ssh}); got != ssh {
		t.Errorf("ssh_config name not found: %+v", got)
	}
	if got, err := Resolve("other.lan", []Connection{fav}); err != nil || got.Host != "other.lan" {
		t.Errorf("unsaved name not parsed as a host: %+v, %v", got, err)
	}
}

func TestParseProtocolFlag(t *testing.T) {
	for in, want := range map[string]string{"": "", "sftp": "SFTP", "FTPS": "FTPS", "Ftp": "FTP"} {
		if got, err := ParseProtocolFlag(in); err != nil || got != want {
			t.Errorf("%q = %q, %v", in, got, err)
		}
	}
	if _, err := ParseProtocolFlag("scp"); err == nil {
		t.Error("scp accepted")
	}
}
