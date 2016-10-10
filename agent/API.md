FORMAT: 1A

# Floe Agent

The Api to a Floe Agent provides access to the floes and history of runs, and some commands to execute floes.

Each floe is a set of connected nodes to describe a task floe.

## Floes [/floes]

### List All Floes [GET]

+ Response 200 (application/json)

        {
            "Floes": [
                {
                    "ID": "test-build",
                    "Name": "Test Build",
                    "Order": 0,
                    "Status": "unknown"
                }
            ]
        }

## Floes [/floes/{floe_id}]

### Summary of Specific Floe [GET]

+ Response 200 (application/json)

        {
            "Message": "OK",
            "Payload": {
                "Floe": {
                    "ID": "test-build",
                    "Name": "Test Build",
                    "Order": 0,
                    "Status": "unknown"
                },
                "Runs": {
                    "FloeID": "test-build",
                    "MaxUsedID": 28,
                    "Summaries": [
                        {
                            "Completed": true,
                            "Duration": 3,
                            "Error": "",
                            "Reason": "",
                            "RunID": 1,
                            "Start": "2016-09-26T23:42:16.247511643Z"
                        },
                        {
                            "Completed": true,
                            "Duration": 3,
                            "Error": "",
                            "Reason": "",
                            "RunID": 28,
                            "Start": "2016-09-26T23:58:12.892471079Z"
                        }
                    ]
                }
            }
        }