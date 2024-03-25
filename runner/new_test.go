package runner

import (
	. "gosh/config"
	"io"
	"os"
	"strings"
	"testing"
)

func testMain(m *testing.M) {
	InitConfig()
	m.Run()
}

func TestLsFunctionality(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	for i := 0; i < 3; i++ {
		file, err := os.CreateTemp(dir, "file")
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = ls(dir)
	if err != nil {
		t.Fatal(err)
	}

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)

	files := strings.Split(strings.Trim(string(out), "\n"), "\n")

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

func TestLsNoArguments(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := ls()
	if err != nil {
		t.Fatal(err)
	}

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)

	if len(strings.Trim(string(out), "\n")) == 0 {
		t.Errorf("Expected some output, got none")
	}
}

func TestLsNonExistentDirectory(t *testing.T) {
	err := ls("/non/existent/directory")
	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestCdFunctionality(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = cd(dir)
	if err != nil {
		t.Fatal(err)
	}

	cwd := AppConfig.AbsolutePath
	want, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	if cwd != want {
		t.Errorf("Expected current directory to be %s, got %s", dir, cwd)
	}
}

func TestCdNoArguments(t *testing.T) {
	err := cd()
	if err != nil {
		t.Fatal(err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if cwd != AppConfig.UserHomeDir {
		t.Errorf("Expected current directory to be %s, got %s", AppConfig.UserHomeDir, cwd)
	}
}

func TestCdNonExistentDirectory(t *testing.T) {
	err := cd("/non/existent/directory")
	if err == nil {
		t.Errorf("Expected an error, got none")
	}
}

func TestLsAndCwd(t *testing.T) {
	dir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	err = cd(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		file, err := os.CreateTemp(dir, "file")
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err = ls()
	if err != nil {
		t.Fatal(err)
	}

	w.Close()
	os.Stdout = oldStdout

	out, _ := io.ReadAll(r)

	files := strings.Split(strings.Trim(string(out), "\n"), "\n")

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}
