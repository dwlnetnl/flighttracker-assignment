# Flight tracker service
This service calculates a flight route based on a list of individual flights.

## Run service
To start the service run `go run ./cmd/flightpathd`, it will start the service
on port `8080`.

## API
The service has the following endpoints:

* `POST /calculate`: calculate a flight route based on a list of individual flights. Input: an array of airport code pairs. Output: an airport pair.

Examples of input and output:
```json
[["SFO", "EWR"]] => ["SFO", "EWR"]
[["ATL", "EWR"], ["SFO", "ATL"]] => ["SFO", "EWR"]
[["IND", "EWR"], ["SFO", "ATL"], ["GSO", "IND"], ["ATL", "GSO"]] => ["SFO", "EWR"]
```
