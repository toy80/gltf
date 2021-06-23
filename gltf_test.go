package gltf

import (
	"os"
	"testing"
)

func TestReadGLTF(t *testing.T) {
	j, err := os.ReadFile("./testdata/bone.gltf")
	if err != nil {
		t.Fatal(err)
	}
	_, err = Decode(j, nil)
	if err != nil {
		t.Fatal(err)
	}
}
