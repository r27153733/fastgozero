package spec_test

import (
	"fmt"

	"github.com/r27153733/fastgozero/tools/fastgoctl/api/spec"
)

func ExampleMember_GetEnumOptions() {
	member := spec.Member{
		Tag: `json:"foo,options=foo|bar|options|123"`,
	}
	fmt.Println(member.GetEnumOptions())
	// Output:
	// [foo bar options 123]
}