package api

import "context"

type LightResponse struct {
	Data []Light `json:"data"`
}

type Light struct {
	ID       string        `json:"id"`
	Metadata LightMetadata `json:"metadata"`
	On       OnState       `json:"on"`
	Dimming  *Dimming      `json:"dimming,omitempty"`
	Color    *Color        `json:"color,omitempty"`
	ColorTemp *ColorTemp   `json:"color_temperature,omitempty"`
}

type LightMetadata struct {
	Name      string `json:"name"`
	Archetype string `json:"archetype"`
}

type OnState struct {
	On bool `json:"on"`
}

type Dimming struct {
	Brightness float64 `json:"brightness"`
}

type Color struct {
	XY XYColor `json:"xy"`
}

type XYColor struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type ColorTemp struct {
	Mirek      *int  `json:"mirek,omitempty"`
	MirekValid bool  `json:"mirek_valid"`
}

type SetLightPayload struct {
	On        *OnState   `json:"on,omitempty"`
	Dimming   *Dimming   `json:"dimming,omitempty"`
	Color     *Color     `json:"color,omitempty"`
	ColorTemp *ColorTemp `json:"color_temperature,omitempty"`
}

func (c *Client) ListLights(ctx context.Context) ([]Light, error) {
	var resp LightResponse
	if err := c.getJSON(ctx, "/resource/light", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (c *Client) SetLight(ctx context.Context, id string, payload SetLightPayload) error {
	_, err := c.put(ctx, "/resource/light/"+id, payload)
	return err
}

func (c *Client) GetLight(ctx context.Context, id string) (*Light, error) {
	var resp LightResponse
	if err := c.getJSON(ctx, "/resource/light/"+id, &resp); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, nil
	}
	return &resp.Data[0], nil
}
