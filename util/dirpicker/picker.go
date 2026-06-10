package dirpicker

import (
	"encoding/base64"
	"os"
	"os/exec"
	"strings"
	"unicode/utf16"
)

// RunDirPicker opens a Windows native folder selection dialog and returns the selected path.
// Returns empty string if user cancels.
func RunDirPicker() (string, error) {
	resultFile, err := os.CreateTemp("", "dirpicker-result-*")
	if err != nil {
		return "", err
	}
	resultFile.Close()
	defer os.Remove(resultFile.Name())

	cmd := buildPickerCmd(resultFile.Name())
	if err := cmd.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(resultFile.Name())
	if err != nil {
		return "", nil // user cancelled or file not written
	}
	return strings.TrimSpace(string(data)), nil
}

// PickerRequest holds the exec.Cmd and result file path for bubbletea integration.
type PickerRequest struct {
	Cmd        *exec.Cmd
	ResultFile string
}

// NewPickerRequest creates a PowerShell folder picker command for use with tea.ExecProcess.
// After the process completes, read the result from ResultFile and clean it up with os.Remove.
func NewPickerRequest() (*PickerRequest, error) {
	resultFile, err := os.CreateTemp("", "dirpicker-result-*")
	if err != nil {
		return nil, err
	}
	resultFile.Close()
	return &PickerRequest{
		Cmd:        buildPickerCmd(resultFile.Name()),
		ResultFile: resultFile.Name(),
	}, nil
}

// ReadResult reads the selected path from the result file and cleans up.
func (r *PickerRequest) ReadResult() (string, error) {
	defer os.Remove(r.ResultFile)
	data, err := os.ReadFile(r.ResultFile)
	if err != nil {
		return "", nil // user cancelled or file not written
	}
	return strings.TrimSpace(string(data)), nil
}

// buildPickerCmd creates a PowerShell command that shows the folder picker and writes result to file.
func buildPickerCmd(resultFile string) *exec.Cmd {
	// PowerShell script: show WinForms FolderBrowserDialog, write path to result file
	script := `Add-Type -AssemblyName System.Windows.Forms` + "\n" +
		`$d = New-Object System.Windows.Forms.FolderBrowserDialog` + "\n" +
		`$d.Description = '请选择文件目录'` + "\n" +
		`$d.ShowNewFolderButton = $false` + "\n" +
		`if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {` + "\n" +
		`  $utf8 = New-Object System.Text.UTF8Encoding($false)` + "\n" +
		`  [System.IO.File]::WriteAllText('` + resultFile + `', $d.SelectedPath, $utf8)` + "\n" +
		`}`

	// Encode as UTF-16LE base64 for -EncodedCommand (avoids all quoting issues)
	utf16Chars := utf16.Encode([]rune(script))
	bytes := make([]byte, len(utf16Chars)*2)
	for i, u := range utf16Chars {
		bytes[i*2] = byte(u)
		bytes[i*2+1] = byte(u >> 8)
	}
	encoded := base64.StdEncoding.EncodeToString(bytes)

	return exec.Command("powershell",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-EncodedCommand", encoded,
	)
}
