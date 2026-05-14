package file

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestGetFiles(t *testing.T) {
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Executable path:", exePath)
	dir, err := os.Getwd()
	fmt.Println("Current directory:", dir)
	if err != nil {
		fmt.Println("Error getting current directory:", err)
		return
	}
	files, err := GetFiles("D:\\goproject\\src\\llm", "pdf")
	if err != nil {
		fmt.Println("Error counting files:", err)
		return
	}
	fmt.Println(files)
}
