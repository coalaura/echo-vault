package main

import "os/exec"

func (e *Echo) Compress() {
	switch e.Extension {
	case "png":
		compressPng(e)
	}
}

func compressPng(e *Echo) {
	_, err := exec.LookPath("pngquant")
	if err != nil {
		return
	}

	cmd := exec.Command("pngquant", "--force", "--ext", ".png", "256", e.Storage())

	_ = cmd.Start()
}
