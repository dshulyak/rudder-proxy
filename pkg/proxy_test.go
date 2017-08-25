package proxy

import "testing"
import "strconv"

func TestAddMeta(t *testing.T) {
	for i, tc := range []struct {
		object   string
		expected bool
	}{
		{
			object: `
metadata:
  name: test-job
  annotations:
    istio.skip: 1
`,
			expected: false,
		},
		{
			object: `
metadata:
  name: test-job
`,
			expected: true,
		},
		{
			object:   "",
			expected: true,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			if rst := addMeta("istio.skip", []byte(tc.object)); rst != tc.expected {
				t.Errorf("result '%t' is different from expected '%t' for object %s", rst, tc.expected, tc.object)
			}
		})
	}
}
