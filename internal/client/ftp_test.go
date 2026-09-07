package client

import "testing"

func TestParseDOSListEntry(t *testing.T) {
	info, ok := parseDOSListEntry("07-20-26  09:52AM       <DIR>          Fuentes")
	if !ok {
		t.Fatal("expected a dir entry to parse")
	}
	if info.Name() != "Fuentes" || !info.IsDir() {
		t.Errorf("got name=%q isDir=%v, want name=Fuentes isDir=true", info.Name(), info.IsDir())
	}

	info, ok = parseDOSListEntry("07-20-26  10:15AM             123456 report.pdf")
	if !ok {
		t.Fatal("expected a file entry to parse")
	}
	if info.Name() != "report.pdf" || info.IsDir() || info.Size() != 123456 {
		t.Errorf("got name=%q isDir=%v size=%d, want name=report.pdf isDir=false size=123456",
			info.Name(), info.IsDir(), info.Size())
	}

	if _, ok := parseDOSListEntry("drwxr-xr-x   8 goftp    20   272 Jul 28 05:03 git-ignored"); ok {
		t.Error("expected a Unix-style LIST line not to match the DOS parser")
	}
}
