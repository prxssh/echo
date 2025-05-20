defmodule Echo.Tracker.UDPClient do
  @moduledoc """
  Client for interacting with UDP trackers.
  """
  @behaviour Echo.Tracker.Behaviour

  alias Echo.Tracker.Types.{Peer, AnnounceAttrs, AnnounceResponse}

  @spec announce(String.t(), AnnounceAttrs.t()) ::
          {:ok, AnnounceResponse.t()} | {:error, String.t()}
  def announce(url, attrs) do
  end
end
