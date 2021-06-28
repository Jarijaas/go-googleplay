module github.com/jarijaas/go-gplayapi

go 1.16

require (
	github.com/ImVexed/muon v0.0.0-20191209030120-589db8f0f250
	github.com/Jarijaas/go-tls-exposed v0.0.0-20201219092535-58270dcefea9
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5 // indirect
	github.com/cheggaaa/pb/v3 v3.0.5
	github.com/chromedp/cdproto v0.0.0-20210526005521-9e51b9051fd0
	github.com/chromedp/chromedp v0.7.3
	github.com/gojektech/heimdall/v6 v6.1.0
	github.com/golang/protobuf v1.4.3
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.1
	github.com/zalando/go-keyring v0.1.0
	golang.org/x/crypto v0.0.0-20201217014255-9d1352758620
	google.golang.org/protobuf v1.25.0
	github.com/jarijaas/goadb v0.0.0-20201208042340-620e0e950ed7
)

replace github.com/jarijaas/goadb v0.0.0-20201208042340-620e0e950ed7 => ../goadb