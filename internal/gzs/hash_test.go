package gzs

import "testing"

func TestHashFileNameWithExtensionNormalizesSnakeBitePaths(t *testing.T) {
	withSlash := HashFileNameWithExtension("/Assets/tpp/pack/mission2/common/mis_com_demo.fpk")
	withoutSlash := HashFileNameWithExtension("Assets/tpp/pack/mission2/common/mis_com_demo.fpk")
	if withSlash != withoutSlash {
		t.Fatalf("leading slash changed hash: %#x != %#x", withSlash, withoutSlash)
	}
	if withSlash>>51 != extensionHashes["fpk"] {
		t.Fatalf("fpk extension bits = %#x, want %#x", withSlash>>51, extensionHashes["fpk"])
	}
}

func TestHashFileNameWithExtensionUsesRecognizedExtensions(t *testing.T) {
	dat := HashFileNameWithExtension("master/0/00.dat")
	xml := HashFileNameWithExtension("metadata.xml")
	if dat>>51 != extensionHashes["dat"] {
		t.Fatalf("dat extension bits = %#x, want %#x", dat>>51, extensionHashes["dat"])
	}
	if xml>>51 != extensionHashes["xml"] {
		t.Fatalf("xml extension bits = %#x, want %#x", xml>>51, extensionHashes["xml"])
	}
}
