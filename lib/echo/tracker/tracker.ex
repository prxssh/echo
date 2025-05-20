defmodule Echo.Tracker do
  @moduledoc """
  The central context for all BitTorrent tracker interactions in Echo.

  `Echo.Tracker` serves as the unified gateway for communicating with both HTTP
  and UDP trackers. It handles announcing and scraping operations, normalizing
  responses into a consistent form, and coordinating retries, error handling,
  and scheduling of subsequent announces. By delegating protocol‐specific logic
  to submodules like `Echo.Tracker.HTTPClient` and `Echo.Tracker.UDPClient`, it
  provides a simple API for the rest of the application to:

    * Register (announce) our peer status with trackers  
    * Query (scrape) tracker statistics (seeders, leechers)  
    * Decode and validate tracker responses  
    * Schedule follow-up announces based on tracker intervals  

  Use this context whenever you need to interact with BitTorrent trackers,
  whether to start a download, update peer counts, or gracefully stop sharing.
  """
  alias Echo.Tracker.{HTTPClient, UDPClient, Types.AnnounceAttrs, Types.AnnounceResponse}

  @spec announce(String.t(), AnnounceAttrs.t()) ::
          {:ok, AnnounceResponse.t()} | {:error, String.t()}
  def announce(url, attrs) do
    case URI.parse(url).scheme do
      scheme when scheme in ["http", "https"] -> HTTPClient.announce(url, attrs)
      "udp" -> UDPClient.announce(url, attrs)
      unknown -> {:error, "tracker: unknown announce scheme '#{unknown}'"}
    end
  end
end
