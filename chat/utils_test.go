package chat

import (
	"testing"
)

func TestEncodeToSha(t *testing.T){
	var str string = "Run Forrest, RUN!"
	if EncodeToSha(str) != "2c26f1825a9c10fa08d63d9ad67215e6b16ab9ed053dec32c8c4966cb1690c69"{
		t.Fatal("Error to Parsing")
	}
}
