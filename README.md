# touchpanel-update-runner

![Circle CI Build Status](https://circleci.com/gh/byuoitav/touchpanel-update-runner/tree/master.svg?style=shield)

### Setup
You'll need the following ENV variables set on your machine to test and develop locally:
```
export FTP_MICROSERVICE_ADDRESS="http://avmetrics1.byu.edu:8002"
export WAIT_FOR_REBOOT_MICROSERVICE_ADDRESS="http://avmetrics1.byu.edu:8003"
export TELNET_MICROSERVICE_ADDRESS="http://avmetrics1.byu.edu:8001"
export TOUCHPANEL_UPDATE_RUNNER_ADDRESS="http://10.5.34.118:8000"
export ELASTICSEARCH_ADDRESS="http://search-byu-oit-av-metrics-ruenjnrqfuhghh7omvtmgcqe7m.us-west-1.es.amazonaws.com/tpupdates/"
```

### Usage
```
{
    "Info": [{
        "Type": [
            "TECHD"
        ],
        "Address": "10.6.36.54",
        "Force": false
    }],
    "HDConfiguration": {
        "Firmware": "https://dl.dropboxusercontent.com/u/9001515/tsx_1.501.0013.004.puf",
        "FirmwareVersion": "1.501.0013",
        "Project": "https://dl.dropboxusercontent.com/u/9001515/TEC_HD.vtz",
        "ProjectDate": "May 11, 2016 22:37:56"
    }
}
```

### Notes
![Whiteboard Picture](whiteboard.jpg)
