package pkg

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	verbose bool
)

func Must(err error) {
	if err != nil {
		fmt.Printf("err: %s\n", err)
		panic(err)
	}
}

func ReadFile(path string) []byte {
	d, err := os.ReadFile(path)
	Must(err)
	return d
}

func OpenNotepadWithFile(path string) {
	cmd := exec.Command("notepad.exe", path)
	err := cmd.Start() 
	Must(err)
}

func Logf(format string, args ...interface{}) {
	if len(args) == 0 {
		fmt.Print(format)
		return
	}
	fmt.Printf(format, args...)
}

func OpenBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	Must(err)
}

func ReadZipFile(path string) map[string][]byte {
	r, err := zip.OpenReader(path)
	Must(err)
	defer r.Close()
	res := map[string][]byte{}
	for _, f := range r.File {
		rc, err := f.Open()
		Must(err)
		d, err := io.ReadAll(rc)
		Must(err)
		rc.Close()
		res[f.Name] = d
	}
	return res
}

func WriteFile(path string, data []byte) {
	err := os.WriteFile(path, data, 0666)
	Must(err)
}

func GetHomeDir() string {
	s, err := os.UserHomeDir()
	Must(err)
	return s
}

func CpFile(dstPath, srcPath string) {
	d, err := os.ReadFile(srcPath)
	Must(err)
	err = os.WriteFile(dstPath, d, 0666)
	Must(err)
}

func runCmd(cmd *exec.Cmd) string {
	if verbose {
		fmt.Printf("> %s\n", strings.Join(cmd.Args, " "))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s failed with '%s'. Output:\n%s\n", strings.Join(cmd.Args, " "), err, string(out))
	}
	Must(err)
	if verbose && len(out) > 0 {
		fmt.Printf("%s\n", out)
	}
	return string(out)
}

func gitStatus(dir string) string {
	cmd := exec.Command("git", "status")
	if dir != "" {
		cmd.Dir = dir
	}
	return runCmd(cmd)
}

func CheckGitClean(dir string) {
	s := gitStatus(dir)
	expected := []string{
		"On branch master",
		"Your branch is up to date with 'origin/master'.",
		"nothing to commit, working tree clean",
	}
	for _, exp := range expected {
		if !strings.Contains(s, exp) {
			fmt.Printf("Git repo in '%s' not clean.\nDidn't find '%s' in output of git status:\n%s\n", dir, exp, s)
			os.Exit(1)
		}
	}
}

func zipAddFile(zw *zip.Writer, zipName string, path string) {
	zipName = filepath.ToSlash(zipName)
	d, err := os.ReadFile(path)
	Must(err)
	w, err := zw.Create(zipName)
	Must(err)
	_, err = w.Write(d)
	Must(err)
	if verbose {
		fmt.Printf("  added %s from %s\n", zipName, path)
	}
}

func zipDirRecur(zw *zip.Writer, baseDir string, dirToZip string) {
	dir := filepath.Join(baseDir, dirToZip)
	files, err := ioutil.ReadDir(dir)
	Must(err)
	for _, fi := range files {
		if fi.IsDir() {
			zipDirRecur(zw, baseDir, filepath.Join(dirToZip, fi.Name()))
		} else if fi.Mode().IsRegular() {
			zipName := filepath.Join(dirToZip, fi.Name())
			path := filepath.Join(baseDir, zipName)
			zipAddFile(zw, zipName, path)
		} else {
			path := filepath.Join(baseDir, fi.Name())
			s := fmt.Sprintf("%s is not a dir or regular file", path)
			panic(s)
		}
	}
}

func CreateZipFile(dst string, baseDir string, toZip ...string) {
	removeFile(dst)
	if len(toZip) == 0 {
		panic("must provide toZip args")
	}
	if verbose {
		fmt.Printf("Creating zip file %s\n", dst)
	}
	w, err := os.Create(dst)
	Must(err)
	defer w.Close()
	zw := zip.NewWriter(w)
	Must(err)
	for _, name := range toZip {
		path := filepath.Join(baseDir, name)
		fi, err := os.Stat(path)
		Must(err)
		if fi.IsDir() {
			zipDirRecur(zw, baseDir, name)
		} else if fi.Mode().IsRegular() {
			zipAddFile(zw, name, path)
		} else {
			s := fmt.Sprintf("%s is not a dir or regular file", path)
			panic(s)
		}
	}
	err = zw.Close()
	Must(err)
}

func removeFile(dst string) {
    if _, err := os.Stat(dst); err == nil {
        err := os.Remove(dst)
        Must(err)
    } else if !os.IsNotExist(err) {
        fmt.Println("zip file doesnt exist !")
        Must(err)
    }
}
