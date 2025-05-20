defmodule Echo.Torrent.Metainfo do
  @moduledoc """
  Module that handles parsing of metainfo into Elixir term.

  Metainfo is a .torrent file that contains metadata related to the torrent.
  All the data in this file is bencoded. The content of a metainfo file is a
  bencoded dictionary. 
  """
  alias Echo.Bencode.Encoder, as: BencodeEncoder

  @type file_t :: %{
          path: [String.t()],
          length: pos_integer()
        }

  @type t :: %__MODULE__{
          name: String.t(),
          files: [file_t()],
          pieces: [binary()],
          info_hash: binary(),
          length: non_neg_integer(),
          encoding: String.t() | nil,
          file_type: :single | :multi,
          announce_urls: [String.t()],
          created_by: String.t() | nil,
          piece_length: non_neg_integer(),
          creation_date: NaiveDateTime.t() | nil
        }

  defstruct(
    name: "",
    length: 0,
    files: [],
    pieces: [],
    info_hash: "",
    encoding: nil,
    file_type: nil,
    created_by: nil,
    piece_length: 0,
    announce_urls: [],
    creation_date: nil
  )

  @doc """
  Transforms a raw metainfo map (as decoded from a `.torrent` file) into a
  `%Metainfo{}` struct. 

  It validates and extracts all required fields—including announce URLs,
  creation date, encoding, file information, piece hashes, and generates the
  SHA-1 info hash—returning `{:ok, metainfo_struct}` on success. If any
  required field is missing or invalid, returns `{:error, reason}`.
  """
  @spec parse(term()) :: {:ok, t()} | {:error, String.t()}
  def parse(metainfo) when is_map(metainfo) do
    with {:ok, announce_urls} <-
           parse_announce_urls(metainfo["announce"], metainfo["announce-list"]),
         creation_date <- parse_creation_date(metainfo["creation date"]),
         created_by <- metainfo["created by"],
         encoding <- metainfo["encoding"],
         {:ok, info_hash} <- generate_info_hash(metainfo["info"]),
         {:ok, info} <- parse_info_dict(metainfo["info"]) do
      {:ok,
       %__MODULE__{
         name: info.name,
         files: info.files,
         encoding: encoding,
         pieces: info.pieces,
         length: info.length,
         info_hash: info_hash,
         created_by: created_by,
         file_type: info.file_type,
         announce_urls: announce_urls,
         creation_date: creation_date,
         piece_length: info.piece_length
       }}
    else
      {:error, reason} -> {:error, reason}
    end
  end

  def parse(other), do: {:error, "metainfo: expected map but got '#{inspect(other)}'"}

  ########## Private

  @spec parse_announce_urls(nil | String.t(), nil | [String.t()]) ::
          {:ok, [String.t()]} | {:error, String.t()}
  defp parse_announce_urls(nil, nil),
    do: {:error, "metainfo: expected announce urls but found none"}

  defp parse_announce_urls(announce, announce_list) do
    {:ok,
     announce_list
     |> List.flatten()
     |> List.insert_at(0, announce)
     |> Enum.uniq()}
  end

  @spec parse_creation_date(nil | pos_integer()) ::
          {:ok, nil | NaiveDateTime.t()} | {:error, String.t()}
  defp parse_creation_date(nil), do: {:ok, nil}

  defp parse_creation_date(epoch) do
    epoch
    |> DateTime.from_unix()
    |> case do
      {:ok, dt} -> DateTime.to_naive(dt)
      {:error, reason} -> {:error, "metainfo: invalid creation date, error: #{inspect(reason)}"}
    end
  end

  @spec generate_info_hash(nil | map()) :: {:ok, binary()} | {:error, String.t()}
  defp generate_info_hash(nil),
    do: {:error, "metainfo: expected info hash dictionary, but got nil"}

  defp generate_info_hash(info_raw) do
    case BencodeEncoder.encode(info_raw) do
      {:ok, encoded} -> {:ok, :crypto.hash(:sha, encoded)}
      {:error, reason} -> {:error, reason}
    end
  end

  @spec parse_info_dict(nil | map()) :: {:ok, map()} | {:error, String.t()}
  defp parse_info_dict(nil), do: {:error, "metainfo: expected info hash dictionary, but got nil"}

  defp parse_info_dict(info) do
    get_field = fn map, key ->
      case Map.get(map, key) do
        nil -> {:error, "metainfo: expected #{key} but got nil"}
        val -> {:ok, val}
      end
    end

    with {:ok, piece_length} <- get_field.(info, "piece length"),
         {:ok, name} <- get_field.(info, "name"),
         {:ok, pieces} <- parse_pieces(info["pieces"]),
         {:ok, files} <- parse_files(info["files"]) do
      length =
        if is_nil(info["length"]),
          do: Enum.reduce(files, 0, fn file, acc -> acc + file.length end),
          else: info["length"]

      {:ok,
       %{
         name: name,
         files: files,
         length: length,
         pieces: pieces,
         piece_length: piece_length,
         file_type: if(Enum.empty?(files), do: :single, else: :multi)
       }}
    else
      {:error, reason} -> {:error, reason}
    end
  end

  @spec parse_pieces(nil | binary()) :: {:ok, [binary()]} | {:error, String.t()}
  defp parse_pieces(nil), do: {:error, "metainfo: expected pieces but got nil"}

  defp parse_pieces(pieces) do
    total = byte_size(pieces)

    if rem(total, 20) == 0 do
      chunks = for <<hash::binary-size(20) <- pieces>>, do: hash
      {:ok, chunks}
    else
      {:error, "metainfo: piece length must be a multiple of 20, got #{total} bytes"}
    end
  end

  @spec parse_files(nil | list()) :: {:ok, [file_t()]} | {:error, String.t()}
  defp parse_files(files) do
    files
    |> List.wrap()
    |> Enum.reduce_while({:ok, []}, fn
      %{"length" => length, "path" => path_list}, {:ok, acc} ->
        cond do
          not is_integer(length) ->
            {:halt, {:error, "metainfo: files length must be integer"}}

          not (is_list(path_list) and Enum.all?(path_list, &is_binary/1)) ->
            {:halt, {:error, "metainfo: file paths must be list of strings"}}

          true ->
            file = %{length: length, path: path_list}
            {:cont, {:ok, [file | acc]}}
        end

      _, _ ->
        {:halt, {:error, "metainfo: files expected list of maps but got '#{inspect(files)}'"}}
    end)
  end
end
