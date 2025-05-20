defmodule Echo.Bencode.EncoderTest do
  use ExUnit.Case, async: true
  alias Echo.Bencode.Encoder

  describe "encode/1" do
    test "encodes integers" do
      assert Encoder.encode(0) == {:ok, "i0e"}
      assert Encoder.encode(42) == {:ok, "i42e"}
      assert Encoder.encode(-7) == {:ok, "i-7e"}
    end

    test "encodes binaries (strings)" do
      assert Encoder.encode("spam") == {:ok, "4:spam"}
      assert Encoder.encode("") == {:ok, "0:"}
    end

    test "encodes atoms by converting to string" do
      assert Encoder.encode(:spam) == {:ok, "4:spam"}
    end

    test "encodes lists, including nested lists" do
      assert Encoder.encode([]) == {:ok, "le"}
      assert Encoder.encode(["spam", "eggs"]) == {:ok, "l4:spam4:eggse"}
      assert Encoder.encode([1, "a", [2]]) == {:ok, "li1e1:ali2eee"}
    end

    test "encodes maps with string keys, sorted lexicographically" do
      assert Encoder.encode(%{}) == {:ok, "de"}

      assert Encoder.encode(%{"cow" => "moo", "spam" => "eggs"}) ==
               {:ok, "d3:cow3:moo4:spam4:eggse"}

      # unsorted initial map should still sort keys
      assert Encoder.encode(%{"b" => "2", "a" => "1"}) == {:ok, "d1:a1:11:b1:2e"}
    end

    test "encodes maps with atom keys by converting to strings" do
      assert Encoder.encode(%{cow: "moo", spam: "eggs"}) ==
               {:ok, "d3:cow3:moo4:spam4:eggse"}
    end

    test "encodes nested maps" do
      assert Encoder.encode(%{a: %{b: "c"}}) == {:ok, "d1:ad1:b1:cee"}
    end

    test "mixed string and atom keys are all strings and sorted" do
      input = %{"bar" => 2, foo: 1}
      assert Encoder.encode(input) == {:ok, "d3:bari2e3:fooi1ee"}
    end

    test "returns error for unsupported types" do
      assert {:error, _} = Encoder.encode(3.14)
      assert {:error, _} = Encoder.encode({:tuple})
    end

    test "returns error for invalid map keys" do
      assert {:error, _} = Encoder.encode(%{1 => "one"})
    end

    test "returns error if any element in list fails to encode" do
      assert {:error, _} = Encoder.encode([:ok, 3.14])
    end
  end

  describe "encode!/1" do
    test "returns binary on success" do
      assert Encoder.encode!(42) == "i42e"
    end

    test "raises on error" do
      assert_raise RuntimeError, fn -> Encoder.encode!(3.14) end
    end
  end
end
