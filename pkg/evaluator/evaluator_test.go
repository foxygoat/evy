package evaluator

import (
	"bytes"
	"strings"
	"testing"

	"foxygo.at/evy/pkg/assert"
)

func TestBasicEval(t *testing.T) {
	in := "a:=1\n print a 2"
	want := "1 2\n"
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(in, fn)
	assert.Equal(t, want, b.String())
}

func TestParseDeclaration(t *testing.T) {
	tests := map[string]string{
		"a:=1":          "1",
		`a:="abc"`:      "abc",
		`a:=true`:       "true",
		`a:= len "abc"`: "3",
	}
	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			in += "\n print a"
			b := bytes.Buffer{}
			fn := func(s string) { b.WriteString(s) }
			Run(in, fn)
			assert.Equal(t, want+"\n", b.String())
		})
	}
}

func TestReturn(t *testing.T) {
	prog := `
func fox:string
    return "š¦"
end

func fox2
    if true
        print "š¦2"
        return
    end
    print "š£"
end

f := fox
print f
print f f
fox2
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "š¦\nš¦ š¦\nš¦2\n"
	assert.Equal(t, want, b.String())
}

func TestReturnScope(t *testing.T) {
	prog := `
f := 1

func fox1:string
    f := "š¦"
    return f
end

func fox2:string
    return fox1
end

print f
f1 := fox1
print f1
f2 := fox2
print f2
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "1\nš¦\nš¦\n"
	assert.Equal(t, want, b.String())
}

func TestBreak(t *testing.T) {
	tests := []string{
		`
while true
    print "š"
    break
end
`, `
while true
    print "š"
    if true
        break
    end
    print "š£"
end
`, `
stop := false
while true
    if stop
        print "š"
        break
    end
    stop = true
end
`, `
continue := true
while true
    if continue
        print "š"
    else
        break
    end
    continue = false
end
`,
	}
	for _, input := range tests {
		b := bytes.Buffer{}
		fn := func(s string) { b.WriteString(s) }
		Run(input, fn)
		want := "š\n"
		assert.Equal(t, want, b.String(), input)
	}
}

func TestAssignment(t *testing.T) {
	prog := `
f1:num
f2:num
f3 := 3
print f1 f2 f3
f1 = 1
print f1 f2 f3
f1 = f3
f2 = f1
f3 = 4
print f1 f2 f3
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "0 0 3\n1 0 3\n3 3 4\n"
	assert.Equal(t, want, b.String())
}

func TestAssignmentAny(t *testing.T) {
	prog := `
func fox:string
    return "š¦"
end

func lol_any:any
    return "š­"
end

f1:any
f2:num
print "1" f1 f2

f1 = f2
print "2" f1 f2

f1 = fox
print "3" f1 f2

f1 = lol_any
print "4" f1

f3 := f1
print "5" f3==f1

f4:any
f4 = f1
print "6" f4==f1
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	wants := []string{
		"1 false 0",
		"2 0 0",
		"3 š¦ 0",
		"4 š­",
		"5 true",
		"6 true",
		"",
	}
	want := strings.Join(wants, "\n")

	assert.Equal(t, want, b.String())
}

func TestIf(t *testing.T) {
	tests := []string{
		`
if true
    print "š"
else
    print "š£"
end
`,
		`
x := "š£"
if true
    x = "š"
end
print x
`,
		`
if false
    print "š£"
else
    if true
        print "š"
    end
end
`,
		`
if true
    if false
        print "š£1"
    else if true
        print "š"
    else if true
        print "š£2"
    else
        print "š£3"
    end
else
    print "š£4"
end
`,
	}
	for _, input := range tests {
		b := bytes.Buffer{}
		fn := func(s string) { b.WriteString(s) }
		Run(input, fn)
		assert.Equal(t, "š\n", b.String(), "input: %s", input)
	}
}

func TestWhile(t *testing.T) {
	input := `
x := true
while x
	print "š­"
	x = false
end

one_more := true
two_more := true
func has_more:bool
	if one_more
		if two_more
			two_more = false
			return true
		else
			one_more = false
			return true
		end
	end
	return false
end

one_more = true
while has_more
	print "š"
end

while has_more
	print "š£"
end
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(input, fn)
	assert.Equal(t, "š­\nš\nš\n", b.String())
}

func TestExpr(t *testing.T) {
	tests := map[string]string{
		"a := 1 + 2 * 2":                    "5",
		"a := (1 + 2) * 2":                  "6",
		"a := (1 + 2) / 2":                  "1.5",
		"a := (1 + 2) / 2 > 1":              "true",
		"a := (1 + 2) / 2 > 1 and 2 == 2*2": "false",
		"a := (1 + 2) / 2 < 1 or 2 == 2*2":  "false",
		"a := (1 + 2) / 2 < 1 or 2 != 2*2":  "true",
		`a := "abc" + "d"`:                  "abcd",
		`a := "abc" + "d" < "efg"`:          "true",
		`a := "abc" + "d" == "abcd"`:        "true",
		`a := "abc" + "d" != "abcd"`:        "false",
		`a := !(1 == 1)`:                    "false",
		`a := -(3 + 5)`:                     "-8",
		`a := -3 +5`:                        "2",
	}
	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			in += "\n print a"
			b := bytes.Buffer{}
			fn := func(s string) { b.WriteString(s) }
			Run(in, fn)
			assert.Equal(t, want+"\n", b.String())
		})
	}
}

func TestArrayLit(t *testing.T) {
	tests := map[string]string{
		"a := [1]":     "[1]",
		"a := []":      "[]",
		"a := [1 2]":   "[1 2]",
		"a := [1 1+1]": "[1 2]",
		`
b := 3
a := [1 1+1 b]`: "[1 2 3]",
		`
func three:num
    return 3
end
a := [1 1+1 (three)]`: "[1 2 3]",
	}
	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			in += "\n print a"
			b := bytes.Buffer{}
			fn := func(s string) { b.WriteString(s) }
			Run(in, fn)
			assert.Equal(t, want+"\n", b.String())
		})
	}
}

func TestIndex(t *testing.T) {
	tests := map[string]string{
		// x := ["a","b","c"]; x = "abc"
		"print x[0]":  "a",
		"print x[1]":  "b",
		"print x[2]":  "c",
		"print x[-1]": "c",
		"print x[-2]": "b",
		"print x[-3]": "a",
		`
		n1 := 1
		print x[n1 - 1] x[1 + n1]
		`: "a c",
		`
		m := {a: "bingo"}
		print m[x[0]]
		`: "bingo",
	}
	for in, want := range tests {
		in, want := in, want
		for _, decl := range []string{`x := ["a" "b" "c"]`, `x := "abc"`} {
			input := decl + "\n" + in
			t.Run(input, func(t *testing.T) {
				b := bytes.Buffer{}
				fn := func(s string) { b.WriteString(s) }
				Run(input, fn)
				assert.Equal(t, want+"\n", b.String())
			})
		}
	}
}

func TestDoubleIndex(t *testing.T) {
	tests := map[string]string{
		`
		x := [ [1 2 3] ([4 5 6]) ]
		x[0][1] = 99
		print x
		`: "[[1 99 3] [4 5 6]]",
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			b := bytes.Buffer{}
			fn := func(s string) { b.WriteString(s) }
			Run(in, fn)
			assert.Equal(t, want+"\n", b.String())
		})
	}
}

func TestIndexErr(t *testing.T) {
	tests := map[string]string{
		// x := ["a","b","c"]; x = "abc"
		"print x[3]":  "ERROR: index 3 out of bounds, should be between -3 and 2",
		"print x[-4]": "ERROR: index -4 out of bounds, should be between -3 and 2",
		`m := {}
		print m[x[1]]`: "ERROR: no value for key b",
	}
	for in, want := range tests {
		in, want := in, want
		for _, decl := range []string{`x := ["a" "b" "c"]`, `x := "abc"`} {
			input := decl + "\n" + in
			t.Run(input, func(t *testing.T) {
				b := bytes.Buffer{}
				fn := func(s string) { b.WriteString(s) }
				Run(input, fn)
				assert.Equal(t, want, b.String())
			})
		}
	}
}

func TestMapLit(t *testing.T) {
	tests := map[string]string{
		"a := {n:1}":                 "{n:1}",
		"a := {}":                    "{}",
		`a := {name:"fox" age:42}`:   "{name:fox age:42}",
		`a := {name:"fox" age:40+2}`: "{name:fox age:42}",
		`
b := 2
a := {name:"fox" age:40+b}`: "{name:fox age:42}",
		`
func three:num
    return 3
end
a := {name:"fox" age:39+(three)}`: "{name:fox age:42}",
	}
	for in, want := range tests {
		in, want := in, want
		t.Run(in, func(t *testing.T) {
			in += "\n print a"
			b := bytes.Buffer{}
			fn := func(s string) { b.WriteString(s) }
			Run(in, fn)
			assert.Equal(t, want+"\n", b.String())
		})
	}
}

func TestDot(t *testing.T) {
	tests := map[string]string{
		// m := {name: "Greta"}
		"print m.name":    "Greta",
		`print m["name"]`: "Greta",
		`s := "name"
		print m[s]`: "Greta",
	}
	for in, want := range tests {
		in, want := in, want
		input := `m := {name: "Greta"}` + "\n" + in
		t.Run(input, func(t *testing.T) {
			b := bytes.Buffer{}
			fn := func(s string) { b.WriteString(s) }
			Run(input, fn)
			assert.Equal(t, want+"\n", b.String())
		})
	}
}

func TestDotErr(t *testing.T) {
	in := `
m := {a:1}
print m.missing_index
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(in, fn)
	want := "ERROR: no value for key missing_index"
	assert.Equal(t, want, b.String())
}

func TestArrayConcatenation(t *testing.T) {
	prog := `
arr1 := [1]
arr2 := arr1
arr3 := arr1 + arr1
arr4 := arr1 + [2]
arr5 := arr1 + []
arr6 := [] + []
print "1 arr1" arr1
print "1 arr2" arr2
print "1 arr3" arr3
print "1 arr4" arr4
print "1 arr5" arr5
print "1 arr6" arr6
print

arr1[0] = 2
print "2 arr1" arr1
print "2 arr2" arr2
print "2 arr3" arr3
print "2 arr4" arr4
print "2 arr5" arr5
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"1 arr1 [1]",
		"1 arr2 [1]",
		"1 arr3 [1 1]",
		"1 arr4 [1 2]",
		"1 arr5 [1]",
		"1 arr6 []",
		"",
		"2 arr1 [2]",
		"2 arr2 [2]",
		"2 arr3 [1 1]",
		"2 arr4 [1 2]",
		"2 arr5 [1]",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestArraySlice(t *testing.T) {
	prog := `
arr := [1 2 3]
print "1" arr[1:3]
print "2" arr[1:]
print "3" arr[1:2]
print "4" arr[1:1]
print "5" arr[:1]
print

arr2 := arr[:]
arr2[0] = 11
print "6" arr arr2
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"1 [2 3]",
		"2 [2 3]",
		"3 [2]",
		"4 []",
		"5 [1]",
		"",
		"6 [1 2 3] [11 2 3]",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestStringSlice(t *testing.T) {
	prog := `
s := "abc"
print "1" s[1:3]
print "2" s[1:]
print "3" s[1:2]
print "4" s[1:1]
print "5" s[:1]
print

s2 := "A" + s[1:]
print "6" s s2
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"1 bc",
		"2 bc",
		"3 b",
		"4 ",
		"5 a",
		"",
		"6 abc Abc",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestForStepRange(t *testing.T) {
	prog := `
for i := range 2
	print "š" i
end
for i := range -1 1
	print "š£" i
end
for i := range 2 6 2
	print "š­" i
end
for i := range 5 3 -1
	print "š¦" i
end
for i := range 3 5 -1
	print "1š£" i
end
for i := range 3 -1 1
	print "2š£" i
end
for i := range 3 -1
	print "3š£" i
end
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"š 0",
		"š 1",
		"š£ -1",
		"š£ 0",
		"š­ 2",
		"š­ 4",
		"š¦ 5",
		"š¦ 4",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestForArray(t *testing.T) {
	prog := `
for x := range [0 1]
	print "š" x
end
for i := range []
	print "š£" i
end
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"š 0",
		"š 1",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestForString(t *testing.T) {
	prog := `
for x := range "abc"
	print "š" x
end
for i := range ""
	print "š£" i
end
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"š a",
		"š b",
		"š c",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestForMap(t *testing.T) {
	prog := `
m := {a:1 b:2}
for x := range m
	print "š" x  m[x]
end
for i := range {}
	print "š£" i
end
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"š a 1",
		"š b 2",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestMap(t *testing.T) {
	prog := `
m1 := {a:1 b:2}
m2 := m1
print "1" m1 m2

m2.a = 10
m1["b"] = 20
print "2" m1 m2

m2.c = 3
m1["d"] = 4
print "3" m1 m2

m4 := {}
m4.a = 1
m4["b"] = 2
print "4" m4

m5 := {}
m5.a = 1
m5.b = {c:99}
//m5.b.c = 2 // parse error: need to cast any to map...
print "5" m5

m6:{}{}num
m6.a = {A :1}
m6.b = {}
m6.b.c = 2
print "6" m6
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"1 {a:1 b:2} {a:1 b:2}",
		"2 {a:10 b:20} {a:10 b:20}",
		"3 {a:10 b:20 c:3 d:4} {a:10 b:20 c:3 d:4}",
		"4 {a:1 b:2}",
		"5 {a:1 b:{c:99}}",
		"6 {a:{A:1} b:{c:2}}",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestMapErr(t *testing.T) {
	in := `
m := {}
m.a = 1
m.b.c = 2
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(in, fn)
	want := "line 4 column 4: field access with '.' expects map type, found any"
	assert.Equal(t, want, b.String())
}

func TestHas(t *testing.T) {
	prog := `
m := {a:1 b:2}
print (has m "a")
print (has m "MISSING")
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"true",
		"false",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestHasErr(t *testing.T) {
	prog := `
has ["a"] "a" // cannot run 'has' on array
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "line 2 column 15: 'has' takes 1st argument of type '{}', found 'string[]'"
	got := b.String()
	assert.Equal(t, want, got)
}

func TestDel(t *testing.T) {
	prog := `
m1 := {a:1 b:2}
m2 := m1
print "1" m1 m2
del m1 "a"
print "2" m1 m2
del m1 "MISSING"
print "3" m1 m2
del m2 "b"
print "4" m1 m2
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := []string{
		"1 {a:1 b:2} {a:1 b:2}",
		"2 {b:2} {b:2}",
		"3 {b:2} {b:2}",
		"4 {} {}",
		"",
	}
	got := strings.Split(b.String(), "\n")
	assert.Equal(t, len(want), len(got), b.String())
	for i := range want {
		assert.Equal(t, want[i], got[i])
	}
}

func TestDelErr(t *testing.T) {
	prog := `
del ["a"] "a" // cannot delete from array
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "line 2 column 15: 'del' takes 1st argument of type '{}', found 'string[]'"
	got := b.String()
	assert.Equal(t, want, got)
}

func TestJoin(t *testing.T) {
	prog := `
print (join [1 true "x"] ", ")
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "1, true, x\n"
	assert.Equal(t, want, b.String())
}

func TestSprint(t *testing.T) {
	prog := `
s := sprint 1 [2] "x"
print (s)
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "1 [2] x\n"
	assert.Equal(t, want, b.String())
}

func TestParamAssign(t *testing.T) {
	prog := `
x := 1
f x
x = x + 1
f x

func f n:num
	n = n*10
	print n x
end`

	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "10 1\n20 2\n"
	assert.Equal(t, want, b.String())
}

func TestAssign2(t *testing.T) {
	prog := `
x := 1
n := x
n = n * 10
print x
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "1\n"
	assert.Equal(t, want, b.String())
}

func TestAssign3(t *testing.T) {
	prog := `
x:num
x = 1
n:num
n = x
n = n * 10
print x
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "1\n"
	assert.Equal(t, want, b.String())
}

func TestSplit(t *testing.T) {
	prog := `
print (split "a, b, c" ", ")
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "[a b c]\n"
	assert.Equal(t, want, b.String())
}

func TestAnyAssignment(t *testing.T) {
	prog := `
a := 1
b:any
b = a
a = 2
print a b
`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "2 1\n"
	assert.Equal(t, want, b.String())
}

func TestCompositeAssignment(t *testing.T) {
	prog := `
n := 1
a := [n n]
m := {n: n}
n = 2
print n a m`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := "2 [1 1] {n:1}\n"
	assert.Equal(t, want, b.String())
}

func TestDemo(t *testing.T) {
	prog := `
move 10 10
line 20 20

x := 12
print "x:" x
if x > 10
    print "š¦ big x"
end`
	b := bytes.Buffer{}
	fn := func(s string) { b.WriteString(s) }
	Run(prog, fn)
	want := `
'move' not yet implemented
'line' not yet implemented
x: 12
š¦ big x
`[1:]
	assert.Equal(t, want, b.String())
}
