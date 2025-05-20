defmodule Echo.Bencode.DecoderTest do
  use ExUnit.Case, async: true
  alias Echo.Bencode.Decoder

  describe "decode/1" do
    test "decodes simple strings" do
      assert Decoder.decode("4:spam") == {:ok, "spam"}
      assert Decoder.decode("0:") == {:ok, ""}
    end

    test "decodes integers" do
      assert Decoder.decode("i0e") == {:ok, 0}
      assert Decoder.decode("i42e") == {:ok, 42}
      assert Decoder.decode("i-42e") == {:ok, -42}
    end

    test "decodes empty and non-empty lists" do
      assert Decoder.decode("le") == {:ok, []}
      assert Decoder.decode("l4:spam4:eggse") == {:ok, ["spam", "eggs"]}
    end

    test "decodes empty and non-empty dicts" do
      assert Decoder.decode("de") == {:ok, %{}}

      assert Decoder.decode("d3:cow3:moo4:spam4:eggse") ==
               {:ok, %{"cow" => "moo", "spam" => "eggs"}}
    end

    test "decodes nested structures" do
      data = "d4:spaml1:a1:bee"
      assert Decoder.decode(data) == {:ok, %{"spam" => ["a", "b"]}}
    end

    test "error on invalid prefix or trailing data" do
      assert {:error, _} = Decoder.decode("x4:spam")
      assert {:error, _} = Decoder.decode("4:spamxyz")
    end

    test "error on truncated string or missing delimiter" do
      assert {:error, _} = Decoder.decode("4:spa")
      assert {:error, _} = Decoder.decode("i42")
      assert {:error, _} = Decoder.decode("4spam")
    end

    test "error on invalid integer format" do
      assert {:error, _} = Decoder.decode("i042e")
      assert {:error, _} = Decoder.decode("i-0e")
    end
  end

  describe "decode!/1" do
    test "returns decoded value or raises on error" do
      assert Decoder.decode!("4:spam") == "spam"
      assert_raise RuntimeError, fn -> Decoder.decode!("4:spa") end
    end
  end
end
