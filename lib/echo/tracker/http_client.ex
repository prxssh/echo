defmodule Echo.Tracker.HTTPClient do
  @moduledoc """
  An HTTP-based BitTorrent tracker client.

  `Echo.Tracker.HTTPClient` implements `Echo.Tracker.Behaviour` and provides a
  standardized way to announce to and scrape from HTTP trackers.
  """
  @behaviour Echo.Tracker.Behaviour

  alias Echo.Bencode.Decoder, as: BencodeDecoder
  alias Echo.Tracker.Types.{Peer, AnnounceAttrs, AnnounceResponse}

  @spec announce(String.t(), AnnounceAttrs.t()) ::
          {:ok, AnnounceResponse.t()} | {:error, String.t()}
  def announce(url, attrs) do
    params =
      attrs
      |> Map.take([
        :left,
        :port,
        :compact,
        :numwant,
        :peer_id,
        :info_hash,
        :uploaded,
        :downloaded
      ])
      |> Map.put_new(:numwant, 50)
      |> Map.put_new(:compact, 1)
      |> Map.put(:event, Atom.to_string(attrs.event))
      |> Map.reject(fn {_, v} -> is_nil(v) end)

    url
    |> Req.get(params: params)
    |> case do
      {:ok, %Req.Response{status: 200, body: body}} -> parse_tracker_announce_response(body)
      {:ok, %Req.Response{body: error}} -> {:error, inspect(error)}
      {:error, reason} -> {:error, reason}
    end
  end

  ########## Private

  @spec parse_tracker_announce_response(binary()) ::
          {:ok, AnnounceResponse.t()} | {:error, String.t()}
  defp parse_tracker_announce_response(binary) do
    case BencodeDecoder.decode(binary) do
      {:ok, %{"failure reason" => reason}} ->
        {:error, reason}

      {:ok, %{"warning message" => reason}} ->
        {:error, reason}

      {:ok, decoded} ->
        {:ok,
         %AnnounceResponse{
           seeders: decoded["complete"],
           interval: decoded["interval"],
           leechers: decoded["incomplete"],
           tracker_id: decoded["trackerid"],
           min_interval: decoded["min interval"],
           peers: parse_tracker_peers(decoded["peers"])
         }}
    end
  end

  @spec parse_tracker_peers(binary() | list()) :: [Peer.t()]
  defp parse_tracker_peers(peers) when is_binary(peers) do
    for <<a::8, b::8, c::8, d::8, hi::8, lo::8 <- peers>> do
      %Peer{
        port: hi * 256 + lo,
        ip: "#{a}.#{b}.#{c}.#{d}"
      }
    end
  end

  defp parse_tracker_peers(peers) when is_list(peers) do
    for %{"ip" => ip, "port" => port, "peer id" => peer_id} <- peers do
      %Peer{ip: ip, port: port, peer_id: peer_id}
    end
  end
end
