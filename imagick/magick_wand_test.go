// Copyright 2013 Herbert G. Fischer. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package imagick

import (
	"fmt"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

var (
	mw *MagickWand
)

func init() {
	Initialize()
}

func TestNewMagickWand(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	mw := NewMagickWand()

	if !mw.IsVerified() {
		t.Fatal("MagickWand not verified")
	}
}

func TestCloningAndDestroying(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	mw := NewMagickWand()
	clone := mw.Clone()
	if !clone.IsVerified() {
		t.Fatal("Unsuccessful clone")
	}

	clone = nil
	runtime.GC()
	time.Sleep(100 * time.Millisecond)

	if !mw.IsVerified() {
		t.Fatal("MagickWand not properly destroyed")
	}
}

func TestQueryConfigureOptions(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	opts := mw.QueryConfigureOptions("*")
	if len(opts) == 0 {
		t.Fatal("QueryConfigureOptions returned an empty array")
	}
	for _, opt := range opts {
		mw.QueryConfigureOption(opt)
	}
}

func TestNonExistingConfigureOption(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	_, err := mw.QueryConfigureOption("4321foobaramps1234")
	if err == nil {
		t.Fatal("Missing error when trying to get non-existing configure option")
	}
}

func TestQueryFonts(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	fonts := mw.QueryFonts("*")
	if len(fonts) == 0 {
		t.Fatal("ImageMagick have not identified a single font in this system")
	}
}

func TestQueryFormats(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	formats := mw.QueryFormats("*")
	if len(formats) == 0 {
		t.Fatal("ImageMagick have not identified a single image format in this system")
	}
}

func TestDeleteImageArtifact(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	mw := NewMagickWand()
	mw.ReadImage(`logo:`)

	if err := mw.DeleteImageArtifact("*"); err != nil {
		t.Fatalf("Error calling DeleteImageArtifact: %s", err.Error())
	}
}

func TestReadImageBlob(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	mw := NewMagickWand()

	// Read an invalid blob
	blob := []byte{}
	if err := mw.ReadImageBlob(blob); err == nil {
		t.Fatal("Expected a failure when passing a zero length blob")
	}

	mw.ReadImage(`logo:`)
	blob = mw.GetImageBlob()

	// Read a valid blob
	if err := mw.ReadImageBlob(blob); err != nil {
		t.Fatal(err.Error())
	}
}

func TestExportImagePixels(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	mw := NewMagickWand()

	var err error
	if err = mw.ReadImage(`logo:`); err != nil {
		t.Fatal("Failed to read internal logo: image")
	}

	var width, height uint = 100, 100
	err = mw.ScaleImage(width, height)
	if err != nil {
		t.Fatalf("Failed scaling image: %s", err.Error())
	}

	tests := []struct {
		x, y    int
		w, h    uint
		pmap    string
		stype   StorageType
		isError bool // If we expect to fail
	}{
		{0, 0, width, height, "RGB", PIXEL_FLOAT, false},   // 0
		{0, 0, width, height, "RGBA", PIXEL_DOUBLE, false}, // 1
		{0, 0, width, height, "R", PIXEL_FLOAT, false},     // 2
		{0, 0, width, height, "GB", PIXEL_FLOAT, false},    // 3
		{0, 1, width, 1, "RGB", PIXEL_DOUBLE, false},       // 4

		{0, 0, width, 0, "RGB", PIXEL_DOUBLE, true},                    // 5
		{0, 0, 0, height, "RGB", PIXEL_DOUBLE, true},                   // 6
		{0, 0, 0, 0, "RGB", PIXEL_DOUBLE, true},                        // 7
		{int(width) + 1, 0, width, height, "RGB", PIXEL_DOUBLE, true},  // 8
		{0, int(height) + 1, width, height, "RGB", PIXEL_DOUBLE, true}, // 9
		{0, int(height) + 1, width, 1, "RGB", PIXEL_DOUBLE, true},      // 10
	}

	for i, tt := range tests {

		val, err := mw.ExportImagePixels(tt.x, tt.y, tt.w, tt.h, tt.pmap, tt.stype)

		if tt.isError {
			if err == nil {
				t.Errorf("[#%d] Expected test to fail, but it did not", i)
			}
			continue
		}

		if err != nil {
			t.Errorf("[#%d] %s", i, err.Error())
			continue
		}

		typ := reflect.TypeOf(val)
		elem := typ.Elem()
		actualKind := elem.Kind()

		var (
			expectKind reflect.Kind
			size       int
		)

		switch tt.stype {

		case PIXEL_FLOAT:
			expectKind = reflect.Float32
			size = len(val.([]float32))

		case PIXEL_DOUBLE:
			expectKind = reflect.Float64
			size = len(val.([]float64))

		default:
			t.Fatalf("[#%d] Unhandled testcase StorageType: %v", i, tt.stype)
		}

		if actualKind != expectKind {
			t.Errorf("[#%d] Expected slice element type %v, got %v",
				i, expectKind, actualKind)
			continue
		}

		expected := int(tt.w) * int(tt.h) * len(tt.pmap)

		if size != expected {
			t.Errorf("[#%d] Expected %q image to have %d pixel vals; Got %d",
				i, tt.pmap, expected, size)
		}

	}
}

func TestGetQuantumDepth(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	name, depth := GetQuantumDepth()
	if name == "" {
		t.Fatal("Depth name returned was an empty string")
	}
	if depth == 0 {
		t.Fatal("Depth value returned was 0")
	}
}

func TestGetQuantumRange(t *testing.T) {
	Initialize()
	defer func(t *testing.T) {
		checkGC(t)
	}(t)
	defer Terminate()

	name, r := GetQuantumRange()
	if name == "" {
		t.Fatal("Depth name returned was an empty string")
	}
	if r == 0 {
		t.Fatal("Range value returned was 0")
	}
}

func checkGC(t *testing.T) {
	if !isImageMagickCleaned() {
		t.Fatal("Some ImageMagick objects are not destroyed", getObjectCountersString())
	}
}

func getObjectCountersString() string {
	str := fmt.Sprintf("\nmagickWandCounter %d\n", atomic.LoadInt64(&magickWandCounter))
	str += fmt.Sprintf("drawingWandCounter %d\n", atomic.LoadInt64(&drawingWandCounter))
	str += fmt.Sprintf("pixelIteratorCounter %d\n", atomic.LoadInt64(&pixelIteratorCounter))
	str += fmt.Sprintf("pixelWandCounter %d\n", atomic.LoadInt64(&pixelWandCounter))

	return str
}

func BenchmarkExportImagePixels(b *testing.B) {
	wand := NewMagickWand()

	wand.ReadImage("logo:")
	wand.ScaleImage(1024, 1024)

	var val interface{}
	var pixels []float32

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		val, _ = wand.ExportImagePixels(0, 0, 1024, 1024, "RGB", PIXEL_FLOAT)
		pixels = val.([]float32)
	}

	b.StopTimer()

	if len(pixels) == 0 {
		b.Fatal("Pixel slice is 0")
	}
}

func BenchmarkImportImagePixels(b *testing.B) {
	wand := NewMagickWand()

	wand.ReadImage("logo:")
	wand.ScaleImage(1024, 1024)

	val, _ := wand.ExportImagePixels(0, 0, 1024, 1024, "RGB", PIXEL_FLOAT)
	pixels := val.([]float32)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		wand.ImportImagePixels(0, 0, 1024, 1024, "RGB", PIXEL_UNDEFINED, pixels)
	}

	b.StopTimer()
}

type testPixelInterfaceValues struct {
	Pixels  interface{}
	Storage StorageType
}

func TestPixelInterfaceToPtr(t *testing.T) {
	Tests := make([]testPixelInterfaceValues, 6)
	Tests[0].Pixels = []byte{0}
	Tests[0].Storage = PIXEL_CHAR
	Tests[1].Pixels = []float64{0}
	Tests[1].Storage = PIXEL_DOUBLE
	Tests[2].Pixels = []float32{0}
	Tests[2].Storage = PIXEL_FLOAT
	Tests[3].Pixels = []int16{0}
	Tests[3].Storage = PIXEL_SHORT
	Tests[4].Pixels = []int32{0}
	Tests[4].Storage = PIXEL_INTEGER
	Tests[5].Pixels = []int64{0}
	Tests[5].Storage = PIXEL_LONG
	for _, value := range Tests {
		_, storageType, err := pixelInterfaceToPtr(value.Pixels)
		if err != nil {
			t.Fatal("Error when passing", reflect.TypeOf(value.Pixels))
		}
		if storageType != value.Storage {
			t.Fatal("Wrong storage type received for", reflect.TypeOf(value.Pixels))
		}
	}

	_, _, err := pixelInterfaceToPtr(32)
	if err == nil {
		t.Fatal("Expected error when passing invalid type")
	}
}
