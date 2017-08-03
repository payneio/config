package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mitchellh/mapstructure"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUnitConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Config Suite")
}

var _ = Describe("config", func() {

	Describe("basic string set", func() {
		Set("key", "value")
		actual := Get("key")

		It("should return set value as a string", func() {
			Expect(actual).Should(Equal("value"))
		})
	})

	Describe("missing string", func() {
		actual := Get("key-not-appearing-in-this-film")
		It("should return empty string", func() {
			Expect(actual).Should(Equal(""))
		})
	})

	Describe("basic int set", func() {
		Set("key", "42")
		actual := GetInt("key")

		It("should return set value as an int", func() {
			Expect(actual).Should(Equal(42))
		})
	})

	Describe("missing int", func() {
		actual := GetInt("key4")
		It("should return zero", func() {
			Expect(actual).Should(Equal(0))
		})
	})

	Describe("nested set", func() {
		Set("log:level", "value")
		actual := Get("log:level")

		It("should return expected value", func() {
			Expect(actual).Should(Equal("value"))
		})
	})

	Describe("deeply nested set", func() {
		Set("a:b:c", "value")
		actual := Get("a:b:c")

		It("should return expected value", func() {
			Expect(actual).Should(Equal("value"))
		})
	})

	Describe("set with JSON", func() {
		SetJSON("key", `{"hair":"black"}`)
		actual := Get("key:hair")

		It("should return set value", func() {
			Expect(actual).Should(Equal("black"))
		})
	})

	Describe("set with invalid JSON", func() {
		err := SetJSON("key", `{"hair":"black"`)

		It("should return error on set", func() {
			Expect(err).ShouldNot(BeNil())
		})
	})

	Describe("set with JSON list", func() {
		SetJSON("people", `[{"person_id":"a", "hair":"black"}, {"person_id":"b", "hair":"red"}]`)
		peopleGoo := GetAny("people")
		type person struct {
			ID   string `mapstructure:"person_id"`
			Hair string
		}
		var peeps []person
		mapstructure.Decode(peopleGoo, &peeps)
		It("should return list", func() {
			Expect(peeps[0].Hair).Should(Equal("black"))
			Expect(peeps[1].Hair).Should(Equal("red"))
		})
	})

	Describe("stripConfigPrefix", func() {

		Context("underscore", func() {
			val, ok := stripConfigPrefix("config_dog")
			It("should strip", func() {
				Expect(val).Should(Equal("dog"))
				Expect(ok).Should(BeTrue())
			})
		})

		Context("double underscore", func() {
			val, ok := stripConfigPrefix("config__dog")
			It("should strip", func() {
				Expect(val).Should(Equal("dog"))
				Expect(ok).Should(BeTrue())
			})
		})

		Context("upper case", func() {
			val, ok := stripConfigPrefix("CONFIG_DOG")
			It("should strip", func() {
				Expect(val).Should(Equal("dog"))
				Expect(ok).Should(BeTrue())
			})
		})

		Context("mixed case, colon", func() {
			val, ok := stripConfigPrefix("conFIG:dog")
			It("should strip", func() {
				Expect(val).Should(Equal("dog"))
				Expect(ok).Should(BeTrue())
			})
		})

		Context("no prefix", func() {
			val, ok := stripConfigPrefix("hat_trick")
			It("should not strip", func() {
				Expect(val).Should(Equal("hat_trick"))
				Expect(ok).Should(BeFalse())
			})
		})

	})

	Describe("normalizeKey", func() {

		Context("double underscores", func() {
			actual := normalizeKey("config__dog__cat")
			It("should normalize", func() {
				Expect(actual).Should(Equal("config:dog:cat"))
			})
		})

		Context("mixed case", func() {
			actual := normalizeKey("config__DOG__cat")
			It("should normalize", func() {
				Expect(actual).Should(Equal("config:dog:cat"))
			})
		})

		Context("colons", func() {
			actual := normalizeKey("config:dog:cat")
			It("should normalize", func() {
				Expect(actual).Should(Equal("config:dog:cat"))
			})
		})

		Context("mixed underscores", func() {
			actual := normalizeKey("CONFIG__FRIDGE__QUERY_SERVICE__FABRIC_ENDPOINT")
			It("should normalize", func() {
				Expect(actual).Should(Equal("config:fridge:query_service:fabric_endpoint"))
			})
		})

	})

	Describe("environment override", func() {

		Context("environment set in config", func() {
			Reset()
			Set("parent:child", "i am the default")
			Set("environment:test:parent:child", "i am the environmental override")
			Set("env", "test")
			setEnvironment()
			actual := Get("parent:child")
			It("should be overriden", func() {
				Expect(actual).Should(Equal("i am the environmental override"))
			})
		})

		Context("component environment set in config", func() {
			Reset()
			Set("parent:child", "i am the default")
			Set("component:a:environment:test:parent:child", "i am the environmental override")
			Set("env", "test")
			setEnvironment()
			setComponent("a")
			actual := Get("parent:child")
			It("should be overriden", func() {
				Expect(actual).Should(Equal("i am the environmental override"))
			})
		})

		Context("component environment overwrite GetAny", func() {
			Reset()
			Set("parent:child", "i am the default")
			Set("parent:foo", "bar")
			Set("component:a:environment:test:parent:child", "i am the environmental override")
			Set("env", "test")
			setEnvironment()
			setComponent("a")
			actual := GetAny("parent")
			It("should be overriden", func() {
				Expect(actual).ShouldNot(BeNil())
				parent, ok := actual.(map[interface{}]interface{})
				Expect(ok).Should(BeTrue())
				Expect(parent["child"]).Should(Equal("i am the environmental override"))
				Expect(parent["foo"]).Should(Equal("bar"))
			})
		})

	})

	Describe("component override", func() {

		Context("component set in config", func() {
			Reset()
			Set("parent:child", "i am the default")
			Set("component:a:parent:child", "i am the component override")
			setComponent("a")
			actual := Get("parent:child")
			It("should be overriden", func() {
				Expect(actual).Should(Equal("i am the component override"))
			})
		})

	})

	Describe("templated variables", func() {

		Context("received a template variable in config", func() {
			os.Setenv("CONFIG_ROOT", "..")
			Set("key", "{ConfigRoot}/file.ext")
			actual := Get("key")
			It("should have template applied", func() {
				expected := filepath.Join("../file.ext")
				Expect(actual).Should(Equal(expected))
			})
		})

		Context("received template variables in nested config", func() {
			os.Setenv("CONFIGROOT", "..")
			SetJSON("people", `[
                {"id":"a", "hair":"black", "file":"{ConfigRoot}/file.ext"},
                {"id":"b", "hair":"red", "file":"{ConfigRoot}/file2.ext"}
            ]`)

			peopleGoo := GetAny("people")
			type person struct {
				ID   string
				Hair string
				File string
			}
			var peeps []person
			mapstructure.Decode(peopleGoo, &peeps)

			actual := peeps[0].File
			It("should have templates applied", func() {
				Expect(actual).Should(Equal("../file.ext"))
			})

			actual2 := peeps[1].File
			It("should have templates applied", func() {
				Expect(actual2).Should(Equal("../file2.ext"))
			})

		})

	})

})
