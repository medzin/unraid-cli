package commands

import (
	"testing"

	"github.com/medzin/unraid-cli/internal/client"
)

func sampleContainer(id, name string, state client.ContainerState) container {
	return container{
		Id:     id,
		Names:  []string{name},
		Image:  "some-image:latest",
		State:  state,
		Status: "Up 2 hours",
	}
}

func sampleContainers() []container {
	return []container{
		sampleContainer("id-1", "plex", client.ContainerStateRunning),
		sampleContainer("id-2", "sonarr", client.ContainerStateRunning),
		sampleContainer("id-3", "radarr", client.ContainerStateExited),
		sampleContainer("id-4", "nginx", client.ContainerStatePaused),
	}
}

func TestFindContainerID(t *testing.T) {
	cases := []struct {
		name       string
		containers []container
		input      string
		wantID     string
		wantErr    bool
	}{
		{"exact match", sampleContainers(), "plex", "id-1", false},
		{"case insensitive", sampleContainers(), "PLEX", "id-1", false},
		{"strips leading slash", []container{{
			Id: "id-slash", Names: []string{"/mycontainer"}, Image: "img",
			State: client.ContainerStateRunning, Status: "Up",
		}}, "mycontainer", "id-slash", false},
		{"matches second name", []container{{
			Id: "id-multi", Names: []string{"primary", "alias"}, Image: "img",
			State: client.ContainerStateRunning, Status: "Up",
		}}, "alias", "id-multi", false},
		{"unknown name", sampleContainers(), "nonexistent", "", true},
		{"empty list", nil, "anything", "", true},
		{"empty names slice", []container{{
			Id: "id-empty", Names: []string{}, Image: "img",
			State: client.ContainerStateRunning, Status: "Up",
		}}, "anything", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, err := findContainerID(tc.containers, tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tc.wantID {
				t.Errorf("expected %s, got %s", tc.wantID, id)
			}
		})
	}
}

func TestFilterContainersByState(t *testing.T) {
	cases := []struct {
		name       string
		containers []container
		showAll    bool
		wantCount  int
	}{
		{"all containers when showAll", sampleContainers(), true, 4},
		{"only running when not showAll", sampleContainers(), false, 2},
		{"empty when no running", []container{
			sampleContainer("id-1", "a", client.ContainerStateExited),
			sampleContainer("id-2", "b", client.ContainerStatePaused),
		}, false, 0},
		{"empty input", nil, false, 0},
		{"empty input showAll", nil, true, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filtered := filterContainersByState(tc.containers, tc.showAll)
			if len(filtered) != tc.wantCount {
				t.Errorf("expected %d containers, got %d", tc.wantCount, len(filtered))
			}
		})
	}
}
