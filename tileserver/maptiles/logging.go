package maptiles

import (
	"fmt"
	seelog "github.com/cihub/seelog"
)

var (
	Ligneous     seelog.LoggerInterface
	LogDirectory string = "log"
	LogLevel     string = "trace"
)

const (
	LOGGING_NAME string = SERVER_NAME + `-` + VERSION
)

func loadLoggingConfig() {
	// https://github.com/cihub/seelog/wiki/Log-levels
	appConfig := `
<seelog minlevel="` + LogLevel + `">
    <outputs formatid="common">
        <filter levels="critical,error,warn">
            <console formatid="stdout"/>
            <file path="` + LogDirectory + `/error.log" formatid="common"/>
        </filter>
        <filter levels="info,debug,trace">
            <console formatid="stdout"/>
            <file path="` + LogDirectory + `/app.log" formatid="common"/>
        </filter>
    </outputs>
    <formats>
        <format id="common"   format="[` + LOGGING_NAME + `] %UTCDate %UTCTime [%LEVEL] %File %FuncShort:%Line %Msg %n" />
        <format id="stdout"   format="[` + LOGGING_NAME + `] %UTCDate %UTCTime [%LEVEL] %File %FuncShort:%Line %Msg %n" />
    </formats>
</seelog>
`

	logger, err := seelog.LoggerFromConfigAsBytes([]byte(appConfig))
	if err != nil {
		fmt.Println(err)
		return
	}

	Ligneous = logger
}

func init() {
	//DisableLog()
	loadLoggingConfig()
}

/*
// DisableLog disables all library log output
func DisableLog() {
	Ligneous = seelog.Disabled
}
*/
