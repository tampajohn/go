// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build js,wasm

package js

import "sync"

var pendingCallbacks = Global.Get("Array").New()

var makeCallbackHelper = Global.Call("eval", `
	(function(id, pendingCallbacks, resolveCallbackPromise) {
		return function() {
			pendingCallbacks.push({ id: id, args: arguments });
			resolveCallbackPromise();
		};
	})
`)

var makeEventCallbackHelper = Global.Call("eval", `
	(function(preventDefault, stopPropagation, stopImmediatePropagation, fn) {
		return function(event) {
			if (preventDefault) {
				event.preventDefault();
			}
			if (stopPropagation) {
				event.stopPropagation();
			}
			if (stopImmediatePropagation) {
				event.stopImmediatePropagation();
			}
			fn(event);
		};
	})
`)

var callbacksMu sync.Mutex
var callbacks = make(map[uint32]func([]Value))
var nextCallbackID uint32 = 1

// Callback is a Go function that got wrapped for use as a JavaScript callback.
// It can be used as an argument with Set, Call, etc.
type Callback struct {
	id    uint32
	value Value
}

// NewCallback returns a wrapped callback which can be used as an argument with Set, Call, etc.
// Invoking the callback in JavaScript will queue the Go function fn for execution.
// This execution happens asynchronously on a special goroutine that handles all callbacks.
// As a consequence, if one callback blocks this goroutine, other callbacks will not be processed.
// Callback.Close must be called to free up resources when the callback will not be used any more.
func NewCallback(fn func(args []Value)) Callback {
	callbacksMu.Lock()
	id := nextCallbackID
	nextCallbackID++
	callbacks[id] = fn
	callbacksMu.Unlock()
	return Callback{
		id:    id,
		value: makeCallbackHelper.Invoke(id, pendingCallbacks, resolveCallbackPromise),
	}
}

// NewEventCallback returns a wrapped callback, just like NewCallback, but the callback expects to have exactly
// one argument, the event. It will synchronously call event.preventDefault, event.stopPropagation and/or
// event.stopImmediatePropagation before queuing the Go function fn for execution.
func NewEventCallback(preventDefault, stopPropagation, stopImmediatePropagation bool, fn func(event Value)) Callback {
	c := NewCallback(func(args []Value) {
		fn(args[0])
	})
	return Callback{
		id:    c.id,
		value: makeEventCallbackHelper.Invoke(preventDefault, stopPropagation, stopImmediatePropagation, c),
	}
}

func (c Callback) Close() {
	callbacksMu.Lock()
	delete(callbacks, c.id)
	callbacksMu.Unlock()
}

func init() {
	go callbackLoop()
}

func callbackLoop() {
	for {
		sleepUntilCallback()
		for {
			cb := pendingCallbacks.Call("shift")
			if cb == Undefined {
				break
			}

			id := uint32(cb.Get("id").Int())
			callbacksMu.Lock()
			f, ok := callbacks[id]
			callbacksMu.Unlock()
			if !ok {
				Global.Get("console").Call("error", "call to closed callback")
				continue
			}

			argsObj := cb.Get("args")
			args := make([]Value, argsObj.Length())
			for i := range args {
				args[i] = argsObj.Index(i)
			}
			f(args)
		}
	}
}

// sleepUntilCallback is defined in the runtime package
func sleepUntilCallback()
