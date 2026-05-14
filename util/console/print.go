package console

import (
	"fmt"
	"llm-util/constant"
)

func Colorful(input, color string) {
	fmt.Println(color + input + constant.Reset)
}
