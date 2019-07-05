package runtime_test

import (
	"testing"
)

func TestClassStmt(t *testing.T) {
	expect(t, `
		class T {}
		out = T.name
	`, nil, "T")

	expect(t, `
		class T {
			get: func(self, key) {
				return self[key]
			}
		}

		inst := T()
		inst.test = "test"
		out = inst.get("test")
	`, nil, "test")

	expect(t, `
		class T {
			init: func(self) {
				self._test = "test"
			},

			get_test: func(self) {
				return self._test
			}
		}

		inst := T()
		inst._test = "other"
		out = format("%s|%s", inst.get_test(), inst._test)
	`, nil, "test|<undefined>")

	expect(t, `
		class T {
			init: func(self) {
				self._p = "test"
			},
			get_private: func(self) {
				return self._p
			} 
		}
		inst := T()
		out = format("%s|%s", inst._p, inst.get_private())
	`, nil, "<undefined>|test")

	expect(t, `
		class Base {
			init: func(self) {
				self._base = "base"
			},
			describe: func(self) {
				return format("Base(_base=%s)", self._base)
			} 
		}
		
		class T: Base {
			init: func(self) {
				self._t = "t"
			},
			describe: func(self) {
				return format("T(_base=%s, _t=%s)", self._base, self._t)
			} 
		}

		inst := T()
		out = format("%s|%s|%s", inst._base, inst._t, inst.describe())
	`, nil, "<undefined>|<undefined>|T(_base=base, _t=t)")
}
