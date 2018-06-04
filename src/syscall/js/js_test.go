// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build js,wasm

package js_test

import (
	"fmt"
	"syscall/js"
	"testing"
)

var dummys = js.Global.Call("eval", `({
	someBool: true,
	someString: "abc\u1234",
	someInt: 42,
	someFloat: 42.123,
	someArray: [41, 42, 43],
	add: function(a, b) {
		return a + b;
	},
})`)

func TestBool(t *testing.T) {
	want := true
	o := dummys.Get("someBool")
	if got := o.Bool(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
	dummys.Set("otherBool", want)
	if got := dummys.Get("otherBool").Bool(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestString(t *testing.T) {
	want := "abc\u1234"
	o := dummys.Get("someString")
	if got := o.String(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
	dummys.Set("otherString", want)
	if got := dummys.Get("otherString").String(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestInt(t *testing.T) {
	want := 42
	o := dummys.Get("someInt")
	if got := o.Int(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
	dummys.Set("otherInt", want)
	if got := dummys.Get("otherInt").Int(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestIntConversion(t *testing.T) {
	testIntConversion(t, 0)
	testIntConversion(t, 1)
	testIntConversion(t, -1)
	testIntConversion(t, 1<<20)
	testIntConversion(t, -1<<20)
	testIntConversion(t, 1<<40)
	testIntConversion(t, -1<<40)
	testIntConversion(t, 1<<60)
	testIntConversion(t, -1<<60)
}

func testIntConversion(t *testing.T, want int) {
	if got := js.ValueOf(want).Int(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestFloat(t *testing.T) {
	want := 42.123
	o := dummys.Get("someFloat")
	if got := o.Float(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
	dummys.Set("otherFloat", want)
	if got := dummys.Get("otherFloat").Float(); got != want {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestUndefined(t *testing.T) {
	dummys.Set("test", js.Undefined)
	if dummys == js.Undefined || dummys.Get("test") != js.Undefined || dummys.Get("xyz") != js.Undefined {
		t.Errorf("js.Undefined expected")
	}
}

func TestNull(t *testing.T) {
	dummys.Set("test1", nil)
	dummys.Set("test2", js.Null)
	if dummys == js.Null || dummys.Get("test1") != js.Null || dummys.Get("test2") != js.Null {
		t.Errorf("js.Null expected")
	}
}

func TestLength(t *testing.T) {
	if got := dummys.Get("someArray").Length(); got != 3 {
		t.Errorf("got %#v, want %#v", got, 3)
	}
}

func TestIndex(t *testing.T) {
	if got := dummys.Get("someArray").Index(1).Int(); got != 42 {
		t.Errorf("got %#v, want %#v", got, 42)
	}
}

func TestSetIndex(t *testing.T) {
	dummys.Get("someArray").SetIndex(2, 99)
	if got := dummys.Get("someArray").Index(2).Int(); got != 99 {
		t.Errorf("got %#v, want %#v", got, 99)
	}
}

func TestCall(t *testing.T) {
	var i int64 = 40
	if got := dummys.Call("add", i, 2).Int(); got != 42 {
		t.Errorf("got %#v, want %#v", got, 42)
	}
	if got := dummys.Call("add", js.Global.Call("eval", "40"), 2).Int(); got != 42 {
		t.Errorf("got %#v, want %#v", got, 42)
	}
}

func TestInvoke(t *testing.T) {
	var i int64 = 40
	if got := dummys.Get("add").Invoke(i, 2).Int(); got != 42 {
		t.Errorf("got %#v, want %#v", got, 42)
	}
}

func TestNew(t *testing.T) {
	if got := js.Global.Get("Array").New(42).Length(); got != 42 {
		t.Errorf("got %#v, want %#v", got, 42)
	}
}

func TestCallback(t *testing.T) {
	c := make(chan struct{})
	cb := js.NewCallback(func(args []js.Value) {
		if got := args[0].Int(); got != 42 {
			t.Errorf("got %#v, want %#v", got, 42)
		}
		c <- struct{}{}
	})
	defer cb.Close()
	js.Global.Call("setTimeout", cb, 0, 42)
	<-c
}

func TestEventCallback(t *testing.T) {
	for _, name := range []string{"preventDefault", "stopPropagation", "stopImmediatePropagation"} {
		c := make(chan struct{})
		cb := js.NewEventCallback(
			name == "preventDefault",
			name == "stopPropagation",
			name == "stopImmediatePropagation",
			func(event js.Value) {
				c <- struct{}{}
			},
		)
		defer cb.Close()

		event := js.Global.Call("eval", fmt.Sprintf("({ called: false, %s: function() { this.called = true; } })", name))
		js.ValueOf(cb).Invoke(event)
		if !event.Get("called").Bool() {
			t.Errorf("%s not called", name)
		}

		<-c
	}
}

func ExampleNewCallback() {
	var cb js.Callback
	cb = js.NewCallback(func(args []js.Value) {
		fmt.Println("button clicked")
		cb.Close() // close the callback if the button will not be clicked again
	})
	js.Global.Get("document").Call("getElementById", "myButton").Call("addEventListener", "click", cb)
}
