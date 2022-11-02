package main

import (
	"context"
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"os"
	"os/signal"
	"syscall"
	"twitter_oracle/common"
	"twitter_oracle/db"
	"twitter_oracle/log"
	"twitter_oracle/stream"
)

var (
	OriginCommandHelpTemplate = `{{.Name}}{{if .Subcommands}} command{{end}}{{if .Flags}} [command options]{{end}} {{.ArgsUsage}}
{{if .Description}}{{.Description}}
{{end}}{{if .Subcommands}}
SUBCOMMANDS:
  {{range .Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
  {{end}}{{end}}{{if .Flags}}
OPTIONS:
{{range $.Flags}}   {{.}}
{{end}}
{{end}}`
)

var app *cli.App
var (
	portFlag = cli.StringFlag{
		Name:  "port",
		Usage: "restful rpc port",
		Value: "8546",
	}
	//dbIPFlag = cli.StringFlag{
	//	Name:  "db",
	//	Usage: "db ip",
	//}
	beaverFlag = cli.StringFlag{
		Name:  "beaver",
		Usage: "auth beaver token",
	}
)

var commandStart = cli.Command{
	Name:  "start",
	Usage: "start twitter oracle",
	Flags: []cli.Flag{
		portFlag,
		beaverFlag,
	},
	Action: Start,
}

func init() {
	app = cli.NewApp()
	app.Version = "v1.0.0"
	app.Commands = []cli.Command{
		commandStart,
	}
	cli.CommandHelpTemplate = OriginCommandHelpTemplate
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func Start(ctx *cli.Context) {
	//load setups
	//port := ""
	//if ctx.IsSet(portFlag.Name) {
	//	port = ctx.String(portFlag.Name)
	//} else {
	//	panic("port unset")
	//}

	if ctx.IsSet(beaverFlag.Name) {
		common.BeaverToken = ctx.String(beaverFlag.Name)
	} else {
		panic("beaver token unset")
	}

	//init and start services
	dbt, err := db.Init()
	if err != nil {
		panic(err)
	}
	log.Info("db connected")

	sub, err := stream.Init(dbt)
	if err != nil {
		panic(err)
	}
	sub.AddDefaultHanler(stream.LoadThoughtHandler)
	err = sub.Start(context.Background())
	if err != nil {
		panic(err)
	}
	log.Info("stream subscriber started")

	//restS := restful.InitRestService(port, dbt)
	//err = restS.Start()
	//if err != nil {
	//	panic(err)
	//}
	log.Info("rest api started")
	waitToExit()
}

func waitToExit() {
	exit := make(chan bool, 0)
	sc := make(chan os.Signal, 1)
	if !signal.Ignored(syscall.SIGHUP) {
		signal.Notify(sc, syscall.SIGHUP)
	}
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sc {
			fmt.Printf("received exit signal:%v", sig.String())
			close(exit)
			break
		}
	}()
	<-exit
}
