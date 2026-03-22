package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/qselle/hue-cli/internal/api"
	"github.com/qselle/hue-cli/internal/auth"
	"github.com/qselle/hue-cli/internal/format"
)

func NewServer() *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "hue-cli",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_lights",
		Description: "List all Hue lights with their current state (on/off, brightness, color).",
	}, makeListLights())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "set_light",
		Description: "Control a Hue light by name or ID. Can set on/off, brightness, and color.",
	}, makeSetLight())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_scenes",
		Description: "List all available Hue scenes.",
	}, makeListScenes())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "activate_scene",
		Description: "Activate a Hue scene by name or ID.",
	}, makeActivateScene())

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_rooms",
		Description: "List all rooms/zones configured on the Hue Bridge.",
	}, makeListRooms())

	return s
}

// --- list_lights ---

type ListLightsInput struct{}

type LightSummary struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	On         bool    `json:"on"`
	Brightness float64 `json:"brightness,omitempty"`
	ColorHex   string  `json:"color_hex,omitempty"`
}

type ListLightsOutput struct {
	Lights []LightSummary `json:"lights"`
}

func makeListLights() func(context.Context, *mcp.CallToolRequest, ListLightsInput) (*mcp.CallToolResult, ListLightsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ListLightsInput) (*mcp.CallToolResult, ListLightsOutput, error) {
		client, err := getClient()
		if err != nil {
			return nil, ListLightsOutput{}, err
		}

		lights, err := client.ListLights(ctx)
		if err != nil {
			return nil, ListLightsOutput{}, err
		}

		summaries := make([]LightSummary, len(lights))
		for i, l := range lights {
			s := LightSummary{
				ID:   l.ID,
				Name: l.Metadata.Name,
				On:   l.On.On,
			}
			if l.Dimming != nil {
				s.Brightness = l.Dimming.Brightness
			}
			if l.Color != nil && l.Dimming != nil {
				s.ColorHex = format.XYToRGBHex(l.Color.XY.X, l.Color.XY.Y, l.Dimming.Brightness)
			}
			summaries[i] = s
		}

		return nil, ListLightsOutput{Lights: summaries}, nil
	}
}

// --- set_light ---

type SetLightInput struct {
	Name       string  `json:"name" jsonschema:"description=Light name or ID"`
	On         *bool   `json:"on,omitempty" jsonschema:"description=Turn light on (true) or off (false)"`
	Brightness float64 `json:"brightness,omitempty" jsonschema:"description=Brightness level 0-100"`
	Color      string  `json:"color,omitempty" jsonschema:"description=Color as hex RGB (e.g. ff0000 for red)"`
}

type SetLightOutput struct {
	Status  string `json:"status"`
	LightID string `json:"light_id"`
}

func makeSetLight() func(context.Context, *mcp.CallToolRequest, SetLightInput) (*mcp.CallToolResult, SetLightOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input SetLightInput) (*mcp.CallToolResult, SetLightOutput, error) {
		client, err := getClient()
		if err != nil {
			return nil, SetLightOutput{}, err
		}

		lightID, err := resolveLightID(ctx, client, input.Name)
		if err != nil {
			return nil, SetLightOutput{}, err
		}

		var payload api.SetLightPayload

		if input.On != nil {
			payload.On = &api.OnState{On: *input.On}
		}

		if input.Brightness > 0 {
			payload.Dimming = &api.Dimming{Brightness: input.Brightness}
		}

		if input.Color != "" {
			x, y, err := format.HexToXY(input.Color)
			if err != nil {
				return nil, SetLightOutput{}, fmt.Errorf("invalid color: %w", err)
			}
			payload.Color = &api.Color{XY: api.XYColor{X: x, Y: y}}
		}

		if err := client.SetLight(ctx, lightID, payload); err != nil {
			return nil, SetLightOutput{}, err
		}

		return nil, SetLightOutput{Status: "ok", LightID: lightID}, nil
	}
}

// --- list_scenes ---

type ListScenesInput struct{}

type SceneSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type ListScenesOutput struct {
	Scenes []SceneSummary `json:"scenes"`
}

func makeListScenes() func(context.Context, *mcp.CallToolRequest, ListScenesInput) (*mcp.CallToolResult, ListScenesOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ListScenesInput) (*mcp.CallToolResult, ListScenesOutput, error) {
		client, err := getClient()
		if err != nil {
			return nil, ListScenesOutput{}, err
		}

		scenes, err := client.ListScenes(ctx)
		if err != nil {
			return nil, ListScenesOutput{}, err
		}

		summaries := make([]SceneSummary, len(scenes))
		for i, s := range scenes {
			summaries[i] = SceneSummary{
				ID:     s.ID,
				Name:   s.Metadata.Name,
				Active: s.Status.Active == "active",
			}
		}

		return nil, ListScenesOutput{Scenes: summaries}, nil
	}
}

// --- activate_scene ---

type ActivateSceneInput struct {
	Name string `json:"name" jsonschema:"description=Scene name or ID"`
}

type ActivateSceneOutput struct {
	Status  string `json:"status"`
	SceneID string `json:"scene_id"`
}

func makeActivateScene() func(context.Context, *mcp.CallToolRequest, ActivateSceneInput) (*mcp.CallToolResult, ActivateSceneOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ActivateSceneInput) (*mcp.CallToolResult, ActivateSceneOutput, error) {
		client, err := getClient()
		if err != nil {
			return nil, ActivateSceneOutput{}, err
		}

		sceneID, err := resolveSceneID(ctx, client, input.Name)
		if err != nil {
			return nil, ActivateSceneOutput{}, err
		}

		if err := client.ActivateScene(ctx, sceneID); err != nil {
			return nil, ActivateSceneOutput{}, err
		}

		return nil, ActivateSceneOutput{Status: "ok", SceneID: sceneID}, nil
	}
}

// --- list_rooms ---

type ListRoomsInput struct{}

type RoomSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Devices int    `json:"devices"`
}

type ListRoomsOutput struct {
	Rooms []RoomSummary `json:"rooms"`
}

func makeListRooms() func(context.Context, *mcp.CallToolRequest, ListRoomsInput) (*mcp.CallToolResult, ListRoomsOutput, error) {
	return func(ctx context.Context, req *mcp.CallToolRequest, input ListRoomsInput) (*mcp.CallToolResult, ListRoomsOutput, error) {
		client, err := getClient()
		if err != nil {
			return nil, ListRoomsOutput{}, err
		}

		rooms, err := client.ListRooms(ctx)
		if err != nil {
			return nil, ListRoomsOutput{}, err
		}

		summaries := make([]RoomSummary, len(rooms))
		for i, r := range rooms {
			summaries[i] = RoomSummary{
				ID:      r.ID,
				Name:    r.Metadata.Name,
				Devices: len(r.Children),
			}
		}

		return nil, ListRoomsOutput{Rooms: summaries}, nil
	}
}

// --- helpers ---

func getClient() (*api.Client, error) {
	cfg, err := auth.GetValidRemoteConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("not authenticated — run 'hue-cli auth' first: %w", err)
	}
	if cfg.IsRemote() {
		return api.NewRemoteClient(cfg.AccessToken), nil
	}
	return api.NewLocalClient(cfg.BridgeIP, cfg.AppKey), nil
}

func resolveLightID(ctx context.Context, client *api.Client, nameOrID string) (string, error) {
	lights, err := client.ListLights(ctx)
	if err != nil {
		return "", fmt.Errorf("listing lights: %w", err)
	}

	for _, l := range lights {
		if l.ID == nameOrID {
			return l.ID, nil
		}
	}

	for _, l := range lights {
		if strings.EqualFold(l.Metadata.Name, nameOrID) {
			return l.ID, nil
		}
	}

	return "", fmt.Errorf("light not found: %s", nameOrID)
}

func resolveSceneID(ctx context.Context, client *api.Client, nameOrID string) (string, error) {
	scenes, err := client.ListScenes(ctx)
	if err != nil {
		return "", fmt.Errorf("listing scenes: %w", err)
	}

	for _, s := range scenes {
		if s.ID == nameOrID {
			return s.ID, nil
		}
	}

	for _, s := range scenes {
		if strings.EqualFold(s.Metadata.Name, nameOrID) {
			return s.ID, nil
		}
	}

	return "", fmt.Errorf("scene not found: %s", nameOrID)
}
