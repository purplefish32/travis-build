package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"github.com/sromku/go-gitter"
	"github.com/urfave/cli"
)

const travisBuildUrl string = "travis-build.claroline.net"
const travisFilesUrlPrefix string = "travis.claroline.net/preview/"
const travisFilesUrlPostfix string = ".tar.gz"

var gitterToken string
var room string

func clarobotSay(phrase string) {
	api := gitter.New(gitterToken)
	err := api.SendMessage(room, phrase)
	if err != nil {
		fmt.Println(err)
	}
}

func printCommand(cmd *exec.Cmd) {
	fmt.Printf("==> Executing: %s\n", strings.Join(cmd.Args, " "))
}

func printError(err error) {
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("==> Error: %s\n", err.Error()))
	}
}

func printOutput(outs []byte) {
	if len(outs) > 0 {
		fmt.Printf("==> Output: %s\n", string(outs))
	}
}

func main() {
	viper.SetConfigName("travis-build")       // name of config file (without extension)
	viper.AddConfigPath("/etc/travis-build/") // path to look for the config file in

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	gitterToken = viper.GetString("GitterToken")
	room = viper.GetString("Room")

	app := cli.NewApp()
	app.Version = "0.0.6"
	app.Name = "travis-build"
	app.Usage = "Manage Claroline Connect Travis Build deployments"
	app.Commands = []cli.Command{
		{
			Name:    "deploy",
			Aliases: []string{"c"},
			Usage:   "Deploy a new Claroline Connect Travis build",
			Action: func(c *cli.Context) error {
				cst := "docker ps --format {{.Names}} | grep " + c.Args().First()
				cmd := exec.Command("bash", "-c", cst)
				var waitStatus syscall.WaitStatus
				if err := cmd.Run(); err != nil {
					// Did the command fail because of an unsuccessful exit code
					if exitError, ok := err.(*exec.ExitError); ok {
						resp, err := http.Get("http://" + travisFilesUrlPrefix + c.Args().First() + travisFilesUrlPostfix)
						if (err == nil) && (resp.StatusCode == 200) {
							fmt.Println("Deploying: ", c.Args().First())
							clarobotSay("I am deploying a travis build here: [" + c.Args().First() + "](http://" + c.Args().First() + "." + travisBuildUrl + "), the build will be up and running in about 30s")
						} else {
							clarobotSay("Sorry dudes and dudettes I cant reach the travis build (" + c.Args().First() + "), it probably does not exist!")
							os.Exit(1)
						}
						cst = "docker run -id -e BUILD=" + c.Args().First() + " -e VIRTUAL_HOST=" + c.Args().First() + "." + travisBuildUrl + " -p 80 -p 9001:9001 -p 9002:9002 --name " + c.Args().First() + " -t claroline"
						if err := exec.Command("bash", "-c", cst).Run(); err != nil {
							fmt.Fprintln(os.Stderr, err)
							clarobotSay("Sorry somthing went wrong deploying the following build: " + c.Args().First())
							os.Exit(1)
						}
						fmt.Println("Successfully deployed the following Travis build: " + c.Args().First())

						waitStatus = exitError.Sys().(syscall.WaitStatus)
						printOutput([]byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
					}
				} else {
					fmt.Println("The Travis build " + c.Args().First() + " is allready deployed")
					clarobotSay("The Travis build " + c.Args().First() + " is allready deployed")
					os.Exit(1)
					waitStatus = cmd.ProcessState.Sys().(syscall.WaitStatus)
					printOutput([]byte(fmt.Sprintf("%d", waitStatus.ExitStatus())))
				}

				return nil
			},
		},
		{
			Name:    "destroy",
			Aliases: []string{"r"},
			Usage:   "Destroy a Blaroline Connect Travis build deployment",
			Action: func(c *cli.Context) error {
				fmt.Println("Destroying: ", c.Args().First())
				cmd := "docker"
				args := []string{"rm", "-f", c.Args().First()}
				if err := exec.Command(cmd, args...).Run(); err != nil {
					fmt.Println(cmd, args)
					fmt.Fprintln(os.Stderr, err)
					clarobotSay("Sorry somthing went wrong nuking the " + c.Args().First() + " Travis build deployment. The deployment probably did not exist!")
					os.Exit(1)
				}
				fmt.Println("Successfully destroyed container: " + c.Args().First())
				clarobotSay("I just nuked a travis build deployment (" + c.Args().First() + ")")
				return nil
			},
		},
		{
			Name:    "list",
			Aliases: []string{"l"},
			Usage:   "List all Claroline Connect Travis build deployments",
			Action: func(c *cli.Context) error {
				var (
					cmdOut []byte
					err    error
				)
				cmd := "docker ps --format \"- {{.Names}} : {{.Status}}\" | grep pr-"
				if cmdOut, err = exec.Command("bash", "-c", cmd).Output(); err != nil {
					fmt.Fprintln(os.Stderr, "There are no running Travis build deployments")
					clarobotSay("There are no running Travis build deployments")
				} else {
					output := string(cmdOut)
					fmt.Println(output)
					clarobotSay("Here is a list of currently deployed Claroline Connect Travis builds:\n" + output + "")
				}

				return nil
			},
		},
	}

	app.Run(os.Args)
}
