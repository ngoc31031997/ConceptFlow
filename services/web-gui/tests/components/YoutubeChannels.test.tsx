import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { YoutubeChannels } from "../../src/components/YoutubeChannels";
import type { YoutubeAccount, YoutubeApp } from "../../src/types";

const APP: YoutubeApp = {
  client_id: "client-a",
  label: "concer-508105",
  project_id: "concer-508105",
  source_file: "client_secret_a.json",
  redirect_ok: true,
  redirect_uri_hint: null,
};

const CHANNEL_ONE: YoutubeAccount = {
  channel_id: "UC_one",
  channel_title: "Kênh Một",
  client_id: "client-a",
  app_label: "concer-508105",
  is_default: true,
  has_caption_scope: true,
};

const CHANNEL_TWO: YoutubeAccount = {
  channel_id: "UC_two",
  channel_title: "Kênh Hai",
  client_id: "client-a",
  app_label: "concer-508105",
  is_default: false,
  has_caption_scope: true,
};

function mockFetch(accounts: YoutubeAccount[], apps: YoutubeApp[]) {
  return vi.fn().mockImplementation((url: string) => {
    const body = url.includes("/apps") ? apps : accounts;
    return Promise.resolve({ ok: true, status: 200, json: async () => body });
  }) as unknown as typeof fetch;
}

function renderChannels(onChange = vi.fn()) {
  render(<YoutubeChannels projectId="p1" onSelectedChannelChange={onChange} />);
  return onChange;
}

describe("YoutubeChannels", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("lists every connected channel instead of a single connected/not flag", async () => {
    global.fetch = mockFetch([CHANNEL_ONE, CHANNEL_TWO], [APP]);
    renderChannels();

    await waitFor(() => expect(screen.getByTestId("youtube-channel-list")).toBeInTheDocument());
    expect(screen.getByText("Kênh Một")).toBeInTheDocument();
    expect(screen.getByText("Kênh Hai")).toBeInTheDocument();
  });

  it("preselects the default channel and reports it upward", async () => {
    global.fetch = mockFetch([CHANNEL_TWO, CHANNEL_ONE], [APP]);
    const onChange = renderChannels();

    // CHANNEL_ONE is second in the response but is the default, so position
    // must not decide the selection.
    await waitFor(() => expect(onChange).toHaveBeenCalledWith("UC_one"));
  });

  it("reports the newly picked channel when the Creator chooses another", async () => {
    global.fetch = mockFetch([CHANNEL_ONE, CHANNEL_TWO], [APP]);
    const onChange = renderChannels();

    await waitFor(() => expect(screen.getByTestId("youtube-channel-list")).toBeInTheDocument());
    fireEvent.click(screen.getByRole("radio", { name: /Kênh Hai/ }));

    await waitFor(() => expect(onChange).toHaveBeenCalledWith("UC_two"));
  });

  it("reports null when nothing is connected, so publishing does not name an empty channel", async () => {
    global.fetch = mockFetch([], [APP]);
    const onChange = renderChannels();

    await waitFor(() => expect(screen.getByTestId("youtube-no-channels")).toBeInTheDocument());
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it("warns which OAuth client is missing the redirect URI, and names the URI to add", async () => {
    const broken: YoutubeApp = {
      ...APP,
      redirect_ok: false,
      redirect_uri_hint: "http://localhost:3000/oauth/youtube/callback",
    };
    global.fetch = mockFetch([], [broken]);
    renderChannels();

    const warning = await screen.findByTestId("youtube-redirect-warning");
    expect(warning).toHaveTextContent("concer-508105");
    expect(warning).toHaveTextContent("http://localhost:3000/oauth/youtube/callback");
  });

  it("skips the app picker when only one OAuth client is configured", async () => {
    global.fetch = mockFetch([], [APP]);
    // jsdom refuses assignment to window.location.href otherwise.
    const location = { href: "" } as Location;
    vi.spyOn(window, "location", "get").mockReturnValue(location);

    renderChannels();
    await waitFor(() => expect(screen.getByTestId("youtube-add-channel")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("youtube-add-channel"));

    expect(screen.queryByTestId("youtube-app-picker")).not.toBeInTheDocument();
    expect(location.href).toContain("app=client-a");
  });

  it("asks which OAuth client to use when several are configured", async () => {
    const second: YoutubeApp = { ...APP, client_id: "client-b", label: "second-project", source_file: "client_secret_b.json" };
    global.fetch = mockFetch([], [APP, second]);
    renderChannels();

    await waitFor(() => expect(screen.getByTestId("youtube-add-channel")).toBeInTheDocument());
    fireEvent.click(screen.getByTestId("youtube-add-channel"));

    expect(await screen.findByTestId("youtube-app-picker")).toBeInTheDocument();
    expect(screen.getByText("second-project")).toBeInTheDocument();
  });

  it("flags a channel connected before force-ssl was requested (CR-015 FR40.2)", async () => {
    const unScoped: YoutubeAccount = { ...CHANNEL_ONE, has_caption_scope: false };
    global.fetch = mockFetch([unScoped, CHANNEL_TWO], [APP]);
    renderChannels();

    await waitFor(() => expect(screen.getByTestId("youtube-channel-list")).toBeInTheDocument());
    expect(screen.getByText("thiếu quyền phụ đề")).toBeInTheDocument();
  });

  it("does not flag a channel that already has the caption scope", async () => {
    global.fetch = mockFetch([CHANNEL_ONE, CHANNEL_TWO], [APP]);
    renderChannels();

    await waitFor(() => expect(screen.getByTestId("youtube-channel-list")).toBeInTheDocument());
    expect(screen.queryByText("thiếu quyền phụ đề")).not.toBeInTheDocument();
  });
});
