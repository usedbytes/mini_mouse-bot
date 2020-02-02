module github.com/usedbytes/thunk-bot

go 1.13

require (
	github.com/BurntSushi/toml v0.3.1
	github.com/gvalkov/golang-evdev v0.0.0-20191114124502-287e62b94bcb
	github.com/jkeiser/iter v0.0.0-20140714165249-67b94d6149c6 // indirect
	github.com/jochenvg/go-udev v0.0.0-20170601065152-b7a82d7b755e
	github.com/usedbytes/battery v0.0.0-20170730185145-863fd3e70fc1 // indirect
	github.com/usedbytes/bno055 v0.0.0-20180915141054-599fcde71dbc
	github.com/usedbytes/bot_matrix/datalink v0.0.0-00010101000000-000000000000
	github.com/usedbytes/input2 v0.0.0-20190127222143-4e961c02eab8
	github.com/usedbytes/linux-led v0.0.0-20190215204929-a48a79538591
	github.com/usedbytes/mini_mouse/cv v0.0.0-00010101000000-000000000000
	github.com/usedbytes/picamera v0.0.0-20190209104458-3ab2614692ba
	gitlab.com/gomidi/midi v1.14.1
	golang.org/x/sys v0.0.0-20200202164722-d101bd2416d5 // indirect
	periph.io/x/periph v3.6.2+incompatible
)

replace github.com/usedbytes/bot_matrix/datalink => github.com/usedbytes/bot_matrix-datalink v0.0.0-20180917185942-df1c01bc955d

replace github.com/usedbytes/mini_mouse/cv => github.com/usedbytes/mini_mouse-cv v0.0.0-20190217141602-6ac9bd79b60f

replace github.com/gvalkov/golang-evdev => github.com/usedbytes/golang-evdev v0.0.0-20171217122358-fb42b6b615fa
