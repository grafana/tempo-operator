package manifestutils

import "fmt"

func Name(tempo string, component string) string {
	return fmt.Sprintf("tempo-%s-%s", tempo, component)
}
