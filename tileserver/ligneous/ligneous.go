package ligneous

import (
	"fmt"
	seelog "github.com/cihub/seelog"
)

// https://github.com/cihub/seelog/wiki/Log-levels

var (
	LogDirectory string = "log"
	//LogLevel      string = "trace"
	LogLevel string = "debug"
)

func InitLogger(prefix string) (seelog.LoggerInterface, error) {
	logging_config := `
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
        <format id="common"   format="[` + prefix + `] %UTCDate %UTCTime [%LEVEL] %File %FuncShort:%Line %Msg %n" />
        <format id="stdout"   format="[` + prefix + `] %UTCDate %UTCTime [%LEVEL] %File %FuncShort:%Line %Msg %n" />
    </formats>
</seelog>
`

	logger, err := seelog.LoggerFromConfigAsBytes([]byte(logging_config))
	if err != nil {
		fmt.Println(err)
		return logger, err
	}

	return logger, nil
}

/*
func DisableLog() {
	NetworkLogger = seelog.Disabled
	ServerLogger = seelog.Disabled
}
*/
