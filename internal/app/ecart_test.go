package app_test

import "testing"

func TestRowShowsTheEcartWithThePreviousSnapshot(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"arr":{}}` + "\n")
	// La source s'est vidée : 40.2 MiB au lieu de 1.3 GiB, en 4 s de moins.
	plakar.ls("arr", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   40.2 MiB       21s /root/docker tags=arr\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad    1.3 GiB       25s /root/docker tags=arr\n" +
		"2026-11-05T23:00:07Z   0b1c2d3e    1.2 GiB       30s /root/docker tags=arr\n"})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["arr"], row{
		"id":               "93ebf1a2",
		"size":             "40.2 MiB",
		"size-ecart":       "−97 %",
		"duration":         "21s",
		"duration-ecart":   "−4s",
		"size-ecart.title": "par rapport au Snapshot précédent a4e9d6ad (1.3 GiB, 25s)",
	})
}

func TestEcartIsSignedAndOnlyShownWithAPreviousSnapshot(t *testing.T) {
	plakar := newFakePlakar(t)
	plakar.stores(`{"seul":{}}` + "\n" + `{"hausse":{}}` + "\n" + `{"stable":{}}` + "\n" + `{"depuis-zero":{}}` + "\n")
	plakar.ls("seul", response{Stdout: "2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       21s /root/docker tags=arr\n"})
	plakar.ls("hausse", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   1.5 KiB     1m11s /data\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad   1.0 KiB       25s /data\n"})
	plakar.ls("stable", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2   1.3 GiB       25s /data\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad   1.3 GiB       25s /data\n"})
	plakar.ls("depuis-zero", response{Stdout: "" +
		"2026-11-06T11:00:05Z   93ebf1a2      3 B        0s /data\n" +
		"2026-11-06T05:00:09Z   a4e9d6ad      0 B        0s /data\n"})
	svc := startService(t, plakar, newClock("2026-11-06T17:00:05Z"), nil)

	_, rows := svc.dashboard()

	assertRow(t, rows["seul"], row{"size": "1.3 GiB", "size-ecart": "", "duration-ecart": ""})
	assertRow(t, rows["hausse"], row{"size-ecart": "+50 %", "duration-ecart": "+46s"})
	assertRow(t, rows["stable"], row{"size-ecart": "0 %", "duration-ecart": "0s"})
	assertRow(t, rows["depuis-zero"], row{"size-ecart": "nouveau", "duration-ecart": "0s"})
}
