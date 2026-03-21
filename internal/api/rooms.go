package api

import "context"

type RoomResponse struct {
	Data []Room `json:"data"`
}

type Room struct {
	ID       string       `json:"id"`
	Metadata RoomMetadata `json:"metadata"`
	Children []ResourceRef `json:"children"`
	Services []ResourceRef `json:"services"`
}

type RoomMetadata struct {
	Name      string `json:"name"`
	Archetype string `json:"archetype"`
}

func (c *Client) ListRooms(ctx context.Context) ([]Room, error) {
	var resp RoomResponse
	if err := c.getJSON(ctx, "/resource/room", &resp); err != nil {
		return nil, err
	}
	return resp.Data, nil
}
