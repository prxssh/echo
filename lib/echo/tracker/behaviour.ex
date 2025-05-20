defmodule Echo.Tracker.Behaviour do
  @moduledoc """
  Defines the behaviour that the modules implementing tracker should adhere to.

  Tracker is a service which responds to requests from clients. The requests
  include metrics from clients that help the tracker keep the overall
  statistics about the torrent. The response includes a peer list that helps
  the client participate in the torrent. The base URL consists of the announce
  url as defined in the metainfo file. The parameters are then added to this
  URL.

  Note: All binary data in the URL (particularly `info_hash` and `peer_id`)
  must be properly escaped.
  """
  alias Echo.Tracker.Types.{AnnounceResponse, AnnounceAttrs}

  @callback announce(url :: String.t(), attrs :: AnnounceAttrs.t()) ::
              {:ok, AnnounceResponse.t()} | {:error, String.t()}
end
