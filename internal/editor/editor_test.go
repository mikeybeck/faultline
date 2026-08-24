package editor

import "testing"

func TestBuildVSCode(t *testing.T) {
	o := Opener{Command: "code"}
	bin, args, err := o.build("/tmp/PaymentController.php", 81)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "code" {
		t.Fatalf("bin = %q", bin)
	}
	if len(args) != 2 || args[0] != "--goto" || args[1] != "/tmp/PaymentController.php:81" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildPhpStorm(t *testing.T) {
	o := Opener{Command: "phpstorm"}
	bin, args, err := o.build("/tmp/User.php", 41)
	if err != nil {
		t.Fatal(err)
	}
	_ = bin
	if len(args) < 3 || args[0] != "--line" || args[1] != "41" {
		t.Fatalf("args = %#v", args)
	}
}

func TestBuildTemplate(t *testing.T) {
	o := Opener{Command: "nvim +{line} {file}"}
	bin, args, err := o.build("/tmp/a.php", 10)
	if err != nil {
		t.Fatal(err)
	}
	if bin != "sh" || len(args) != 2 {
		t.Fatalf("bin/args = %s %#v", bin, args)
	}
	if args[1] != "nvim +10 '/tmp/a.php'" {
		t.Fatalf("cmd = %q", args[1])
	}
}
