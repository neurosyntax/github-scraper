package parse

/*import "testing"

func Test(t *testing.T) {
	var tests []struct {
		s, want string
	}{
		{"",""},
		{"public int myMethod(int x, int y)", ""}
	}
}*/

import "fmt"

func main() {
	fmt.Println(parseFuncHeader("public int myfunc(int x, int y)"))
}