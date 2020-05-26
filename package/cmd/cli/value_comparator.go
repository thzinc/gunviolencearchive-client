package cli

import (
	"fmt"
	"strings"

	"github.com/thzinc/gunviolencearchive-client/package/gvaclient"
)

type comparator struct {
	value gvaclient.Comparator
}

func (c *comparator) Value() gvaclient.Comparator {
	return c.value
}

func (c *comparator) String() string {
	return string(c.value)
}

func (c *comparator) Set(s string) error {
	switch gvaclient.Comparator(s) {
	case
		gvaclient.IsEqualTo,
		gvaclient.IsGreaterThan,
		gvaclient.IsLessThan,
		gvaclient.IsNotEqualTo:
		c.value = gvaclient.Comparator(s)
		return nil
	default:
		allowedValues := strings.Join([]string{
			string(gvaclient.IsEqualTo),
			string(gvaclient.IsGreaterThan),
			string(gvaclient.IsLessThan),
			string(gvaclient.IsNotEqualTo),
		}, ", ")

		return fmt.Errorf("comparator must be one of: %s", allowedValues)
	}
}

func (c *comparator) Type() string {
	return "Comparator"
}
