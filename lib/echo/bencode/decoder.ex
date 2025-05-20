defmodule Echo.Bencode.Decoder do
  @moduledoc """
  Bencoding is a way to specify and organize data in a terse format. It
  supports the following types: byte strings, integeres, lists, and
  dictionaries.

  BitTorrent uses bencoding for .torrent files to represent metadata about
  files and trackers, including info dictionaries, file lists, piece hashes and
  tracker URLs.

  This module provides function to decode bencoded data into Elixir data
  structures.
  """
  @type decode_result_t :: integer() | String.t() | list() | map()

  @doc "Same as `decode/1` but raises upon failure."
  @spec decode!(binary()) :: decode_result_t()
  def decode!(data) do
    case decode(data) do
      {:ok, result} -> result
      {:error, reason} -> raise reason
    end
  end

  @doc """
  Decode the given bencoded binary, returning the decoded Elixir term.
  """
  @spec decode(binary()) :: {:ok, decode_result_t()} | {:error, String.t()}
  def decode(data) do
    case do_decode(data) do
      {:ok, result, ""} ->
        {:ok, result}

      {:ok, _, rest} ->
        {:error, "bencode: trailing data after bencoded value: #{inspect(rest)}"}

      {:error, reason} ->
        {:error, reason}
    end
  end

  ########## Private

  @spec do_decode(binary()) :: {:ok, decode_result_t(), binary()} | {:error, String.t()}
  defp do_decode(<<"d", rest::binary>>), do: decode_dict(rest)

  defp do_decode(<<"l", rest::binary>>), do: decode_list(rest)

  defp do_decode(<<"i", _::binary>> = data), do: decode_int(data)

  defp do_decode(<<c, _::binary>> = data) when c in ?0..?9, do: decode_string(data)

  defp do_decode(<<>>), do: {:error, "bencode: unexpected EOI"}

  defp do_decode(<<other, _::binary>>), do: {:error, "bencode: invalid prefix: #{inspect(other)}"}

  @spec decode_dict(binary(), map()) :: {:ok, map(), binary()} | {:error, String.t()}
  defp decode_dict(data, acc \\ Map.new())

  defp decode_dict(<<"e", rest::binary>>, acc), do: {:ok, acc, rest}

  defp decode_dict(data, acc) do
    with {:ok, key, rest} <- do_decode(data),
         {:ok, value, rest} <- do_decode(rest) do
      decode_dict(rest, Map.put(acc, key, value))
    else
      {:error, reason} -> {:error, reason}
    end
  end

  @spec decode_list(binary(), list()) :: {:ok, list(), binary()} | {:error, String.t()}
  defp decode_list(data, acc \\ [])

  defp decode_list(<<"e", rest::binary>>, acc), do: {:ok, Enum.reverse(acc), rest}

  defp decode_list(data, acc) do
    case do_decode(data) do
      {:ok, val, rest} -> decode_list(rest, [val | acc])
      {:error, reason} -> {:error, reason}
    end
  end

  @spec decode_int(binary()) :: {:ok, integer(), binary()} | {:error, String.t()}
  defp decode_int(<<"i", rest::binary>>), do: parse_integer(rest, "e")

  defp decode_int(data),
    do: {:error, "bencode: invalid integer prefix, expected 'i', got: #{inspect(data)}"}

  @spec decode_int(binary()) :: {:ok, String.t(), binary()} | {:error, String.t()}
  defp decode_string(data) do
    case parse_integer(data, ":") do
      {:ok, len, rest} -> parse_string(rest, len)
      {:error, reason} -> {:error, reason}
    end
  end

  defp parse_string(_, len) when len < 0,
    do: {:error, "bencode: invalid string, length can't be negative"}

  defp parse_string(data, len) do
    if byte_size(data) < len do
      {:error, "bencode: truncated string, expected #{len} bytes"}
    else
      <<value::binary-size(len), rest::binary>> = data
      {:ok, value, rest}
    end
  end

  @spec parse_integer(binary(), String.t()) :: {:ok, integer(), binary()} | {:error, String.t()}
  defp parse_integer(data, delim) do
    case :binary.split(data, delim) do
      [int_str, rest] ->
        if valid_int_str?(int_str) do
          {int, _} = Integer.parse(int_str)
          {:ok, int, rest}
        else
          {:error, "bencode: invalid integer '#{inspect(int_str)}'"}
        end

      _ ->
        {:error, "bencode: missing '#{delim}' delimiter"}
    end
  end

  defp valid_int_str?(<<"-", rest::binary>>),
    do: rest != "" and not String.starts_with?(rest, "0") and String.match?(rest, ~r/^\d+$/)

  defp valid_int_str?(str) when is_binary(str),
    do: str == "0" or (not String.starts_with?(str, "0") and String.match?(str, ~r/^\d+$/))
end
