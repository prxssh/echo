defmodule Echo.Torrent do
  @moduledoc """
  The single context for all torrent-related workflows in Echo.

  `Echo.Torrent` serves as the central gateway for handling `.torrent` files
  and their metadata throughout the application. It encapsulates and
  coordinates the various steps involved in processing torrent data— from
  low-level bencode decoding to rich metadata extraction— by delegating to
  specialized submodules.
  """
  alias Echo.Torrent.Metainfo
  alias Echo.Bencode.Decoder, as: BencodeDecoder

  @doc """
  Reads the `.torrent` file at `path` and parses it into a metainfo struct.
  """
  @spec read(String.t()) :: {:ok, Metainfo.t()} | {:error, String.t()}
  def read(path) do
    with {:ok, data} <- File.read(path),
         {:ok, decoded} <- BencodeDecoder.decode(data),
         {:ok, metainfo} <- Metainfo.parse(decoded) do
      {:ok, metainfo}
    else
      {:error, reason} -> {:error, reason}
    end
  end
end
