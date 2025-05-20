defmodule Echo.Bencode.Encoder do
  @moduledoc """
  Module that handles encoding of Elixir term to bencoded format. It supports
  the following data types: atom, string, list, and map.
  """
  @doc """
  Same as `encode/1` but raises on error
  """
  @spec encode!(term()) :: String.t()
  def encode!(term) do
    case encode(term) do
      {:ok, encoded} -> encoded
      {:error, reason} -> raise reason
    end
  end

  @doc """
  Encode elixir term to bencoded format.

  Supported data types are: atom, string, list, and map.
  """
  @spec encode(atom() | String.t() | list() | map()) :: {:ok, String.t()} | {:error, String.t()}
  def encode(term) do
    case do_encode(term) do
      {:ok, encoded} -> {:ok, encoded}
      {:error, reason} -> {:error, reason}
    end
  end

  ########## Private

  @spec do_encode(term()) :: {:ok, String.t()} | {:error, String.t()}
  defp do_encode(integer) when is_integer(integer),
    do: {:ok, "i" <> Integer.to_string(integer) <> "e"}

  defp do_encode(string) when is_binary(string),
    do: {:ok, (string |> byte_size() |> Integer.to_string()) <> ":" <> string}

  defp do_encode(atom) when is_atom(atom), do: atom |> Atom.to_string() |> do_encode()

  defp do_encode(list) when is_list(list) do
    list
    |> Enum.reduce_while({:ok, []}, fn elem, {:ok, acc} ->
      case do_encode(elem) do
        {:ok, enc} -> {:cont, {:ok, [enc | acc]}}
        error -> {:halt, error}
      end
    end)
    |> case do
      {:ok, encoded} -> {:ok, "l" <> (encoded |> Enum.reverse() |> List.to_string()) <> "e"}
      error -> error
    end
  end

  defp do_encode(map) when is_map(map) do
    case Enum.all?(map, fn {k, _} -> is_binary(k) or is_atom(k) end) do
      true ->
        map
        |> Enum.sort_by(fn {k, _} -> to_string(k) end)
        |> Enum.reduce_while({:ok, ""}, fn {k, v}, {:ok, acc} ->
          with {:ok, encoded_key} <- do_encode(k),
               {:ok, encoded_val} <- do_encode(v) do
            {:cont, {:ok, acc <> encoded_key <> encoded_val}}
          else
            error -> {:halt, error}
          end
        end)
        |> case do
          {:ok, dict} -> {:ok, "d" <> dict <> "e"}
          error -> error
        end

      false ->
        {:error, "bencode: invalid map entries"}
    end
  end

  defp do_encode(other), do: {:error, "bencode: unsupported type #{inspect(other)}"}
end
