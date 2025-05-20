defmodule Echo.Tracker.UDPClient do
  @moduledoc """
  An UDP-based BitTorrent tracker client.

  This module implements `Echo.Tracker.Behaviour` and provides a standardized
  way to announce to and scrape from HTTP trackers.
  """
  @behaviour Echo.Tracker.Behaviour

  require Logger
  import Bitwise
  alias Echo.Tracker.Types.{Peer, AnnounceAttrs, AnnounceResponse}

  @action_connect 0
  @action_announce 1
  @action_scrape 2
  @action_error 3

  @max_retries 8
  @protocol_id 0x41727101980
  @response_timeout :timer.seconds(15)

  @spec announce(String.t(), AnnounceAttrs.t()) ::
          {:ok, AnnounceResponse.t()} | {:error, String.t()}
  def announce(url, attrs) do
    %URI{host: host, port: port} = URI.parse(url)

    with {:ok, socket} <- :gen_udp.open(0, [:binary, active: false, reuseaddr: true]),
         {:ok, ipaddr} <- resolve_host(host),
         {:ok, connection_id} <- do_connect(socket: socket, ipaddr: ipaddr, port: port),
         {:ok, response} <-
           do_announce([socket: socket, ipaddr: ipaddr, port: port], connection_id, attrs) do
      :gen_udp.close(socket)
      {:ok, response}
    else
      {:error, reason} -> {:error, reason}
    end
  end

  ########## Private

  @typep connect_attrs_t :: [
           socket: :gen_udp.socket(),
           ipaddr: :inet.ip_address(),
           port: pos_integer()
         ]

  @spec resolve_host(String.t()) :: {:ok, :inet.ip_address()} | {:error, term()}
  defp resolve_host(host) do
    host
    |> to_charlist()
    |> :inet.getaddr(:inet)
  end

  @spec do_connect(connect_attrs_t(), pos_integer()) :: {:ok, binary()} | {:error, String.t()}
  defp do_connect(connect_attrs, attempt \\ 1)

  defp do_connect(_, attempt) when attempt > @max_retries,
    do: {:error, "[tracker] connect failed, exhausted all attempts"}

  defp do_connect(connect_attrs, attempt) do
    Logger.debug(
      "[tracker] attempting connect, ip: #{inspect(connect_attrs[:ipaddr])}, port: #{connect_attrs[:port]}, attempt: #{attempt}"
    )

    transaction_id = :crypto.strong_rand_bytes(4)
    packet = build_packet(@action_connect, transaction_id)
    response_timeout = @response_timeout * (1 <<< attempt)

    :ok =
      :gen_udp.send(connect_attrs[:socket], connect_attrs[:ipaddr], connect_attrs[:port], packet)

    case :gen_udp.recv(connect_attrs[:socket], 0, response_timeout) do
      {:ok, {_, _, <<@action_connect::32, ^transaction_id::binary-size(4), connection_id::64>>}} ->
        {:ok, connection_id}

      {:ok, {_, _, <<@action_error::32, ^transaction_id::binary-size(4), error::binary>>}} ->
        {:error, "[tracker] udp connect error: #{inspect(error)}"}

      {:error, :timeout} ->
        do_connect(connect_attrs, attempt + 1)

      {:error, reason} ->
        {:error, "[tracker] udp connect failed with error: #{inspect(reason)}"}
    end
  end

  @spec do_announce(connect_attrs_t(), binary(), AnnounceAttrs.t(), pos_integer()) ::
          {:ok, binary()} | {:error, String.t()}
  defp do_announce(connect_attrs, connection_id, announce_attrs, attempt \\ 1)

  defp do_announce(_, _, _, attempt) when attempt > @max_retries,
    do: {:error, "[tracker] announce failed, exhausted all attempts"}

  defp do_announce(connect_attrs, connection_id, announce_attrs, attempt) do
    Logger.debug(
      "[tracker] attempting announce, ip: #{inspect(connect_attrs[:ipaddr])}, port: #{connect_attrs[:port]}, attempt: #{attempt}"
    )

    transaction_id = :crypto.strong_rand_bytes(4)
    packet = build_packet(@action_announce, connection_id, transaction_id, announce_attrs)
    response_timeout = @response_timeout * (1 <<< attempt)

    :ok =
      :gen_udp.send(connect_attrs[:socket], connect_attrs[:ipaddr], connect_attrs[:port], packet)

    case :gen_udp.recv(connect_attrs[:socket], 0, response_timeout) do
      {:ok,
       {_, _,
        <<@action_announce::32, ^transaction_id::binary-size(4), interval::32, leechers::32,
          seeders::32, peers_bin::binary>>}} ->
        {:ok,
         %AnnounceResponse{
           seeders: seeders,
           interval: interval,
           leechers: leechers,
           peers: parse_peers(peers_bin)
         }}

      {:ok, {_, _, <<@action_error::32, ^transaction_id::binary-size(4), error::binary>>}} ->
        {:error, "tracker: udp announce error: #{inspect(error)}"}

      {:error, :timeout} ->
        do_announce(connect_attrs, connection_id, announce_attrs, attempt + 1)

      {:error, reason} ->
        {:error, "tracker: udp announce failed with error: #{inspect(reason)}"}
    end
  end

  defp build_packet(@action_connect, transaction_id),
    do: <<@protocol_id::64, @action_connect::32, transaction_id::binary-size(4)>>

  defp build_packet(@action_announce, connection_id, transaction_id, attrs) do
    key = :crypto.strong_rand_bytes(4)
    {ip, num_want} = {attrs[:ip] || 0, attrs[:numwant] || 50}
    event_code = to_event_code(attrs[:event])

    <<
      connection_id::64,
      @action_announce::32,
      transaction_id::bytes-size(4),
      attrs.info_hash::bytes-size(20),
      attrs.peer_id::bytes-size(20),
      attrs.downloaded::64,
      attrs.left::64,
      attrs.uploaded::64,
      event_code::32,
      ip::32,
      key::bytes-size(4),
      num_want::32,
      attrs.port::16
    >>
  end

  defp to_event_code(:completed), do: 1
  defp to_event_code(:started), do: 2
  defp to_event_code(:stopped), do: 3
  defp to_event_code(_), do: 0

  @spec parse_peers(binary()) :: [Peer.t()]
  defp parse_peers(peers) do
    peers
    |> :binary.bin_to_list()
    |> Enum.chunk_every(6)
    |> Enum.map(fn peer ->
      <<ip::32, port::16>> = :binary.list_to_bin(peer)
      parsed_ip = ip |> :binary.encode_unsigned() |> :binary.bin_to_list() |> List.to_tuple()

      %Peer{ip: parsed_ip, port: port}
    end)
  end
end
