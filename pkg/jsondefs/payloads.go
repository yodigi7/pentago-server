package jsondefs

import "github.com/yodigi7/pentago"

type InitialConnection struct {
	GameId string `json:"gameId"`
}

type InitialConnectionResponse struct {
	ColorNumber pentago.Space `json:"colorNumber"`
}

type MarblePlacement struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type Rotation struct {
	Quadrant  pentago.Quadrant          `json:"quadrant"`
	Direction pentago.RotationDirection `json:"direction"`
}

type Turn struct {
	GameId          string          `json:"gameId"`
	MarblePlacement MarblePlacement `json:"marblePlacement"`
	Rotation        Rotation        `json:"rotation"`
}

type GeneralResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func ToGeneralResponse(code int, msg string) GeneralResponse {
	return GeneralResponse{
		Code:    code,
		Message: msg,
	}
}

func To200Response(msg string) GeneralResponse {
	return ToGeneralResponse(200, msg)
}

func To400Response(msg string) GeneralResponse {
	return ToGeneralResponse(400, msg)
}
