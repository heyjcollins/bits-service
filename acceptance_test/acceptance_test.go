package acceptance_test

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	acceptance "github.com/cloudfoundry-incubator/bits-service/acceptance_test"
	"github.com/cloudfoundry-incubator/bits-service/httputil"
	. "github.com/cloudfoundry-incubator/bits-service/testutil"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	session *gexec.Session
	client  *http.Client
)

func TestEndToEnd(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)

	BeforeSuite(func() {
		err := os.MkdirAll("/tmp/eirinifs/assets", 0755)
		Ω(err).ShouldNot(HaveOccurred())
		file, err := os.Create("/tmp/eirinifs/assets/eirinifs.tar")
		Ω(err).ShouldNot(HaveOccurred())
		file.Close()

		session = acceptance.StartServer("config.yml")
		client = acceptance.CreateTLSClient("ca_cert")
	})

	AfterSuite(func() {
		if session != nil {
			session.Kill()
		}
		gexec.CleanupBuildArtifacts()
		os.Remove("/tmp/eirinifs/assets/eirinifs.tar")
	})

	ginkgo.RunSpecs(t, "EndToEnd HTTPS")
}

var _ = Describe("Accessing the bits-service", func() {
	Context("through private host", func() {
		It("return http.StatusNotFound for a package that does not exist", func() {
			Expect(client.Get("https://internal.127.0.0.1.nip.io:4443/packages/notexistent")).
				To(WithTransform(GetStatusCode, Equal(http.StatusNotFound)))
		})

		It("return http.StatusOK for a package that does exist", func() {
			request, e := httputil.NewPutRequest("https://internal.127.0.0.1.nip.io:4443/packages/myguid", map[string]map[string]io.Reader{
				"package": map[string]io.Reader{"somefilename": CreateZip(map[string]string{"somefile": "lalala\n\n"})},
			})
			Expect(e).NotTo(HaveOccurred())

			Expect(client.Do(request)).To(WithTransform(GetStatusCode, Equal(201)))

			Expect(client.Get("https://internal.127.0.0.1.nip.io:4443/packages/myguid")).
				To(WithTransform(GetStatusCode, Equal(http.StatusOK)))
		})
	})

	Context("through public host", func() {
		It("returns http.StatusForbidden when accessing package through public host without signature", func() {
			Expect(client.Get("https://public.127.0.0.1.nip.io:4443/packages/notexistent")).
				To(WithTransform(GetStatusCode, Equal(http.StatusForbidden)))
		})

		Context("After retrieving a signed URL", func() {
			It("returns http.StatusOK when accessing package through public host with signature", func() {
				request, e := httputil.NewPutRequest("https://internal.127.0.0.1.nip.io:4443/packages/myguid", map[string]map[string]io.Reader{
					"package": map[string]io.Reader{"somefilename": CreateZip(map[string]string{"somefile": "lalala\n\n"})},
				})
				Expect(e).NotTo(HaveOccurred())

				Expect(client.Do(request)).To(WithTransform(GetStatusCode, Equal(201)))

				response, e := client.Do(
					newGetRequest("https://internal.127.0.0.1.nip.io:4443/sign/packages/myguid", "the-username", "the-password"))
				Ω(e).ShouldNot(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))

				signedUrl, e := ioutil.ReadAll(response.Body)
				Ω(e).ShouldNot(HaveOccurred())
				response, e = client.Get(string(signedUrl))
				Ω(e).ShouldNot(HaveOccurred())

				responseBody, e := ioutil.ReadAll(response.Body)
				Expect(e).NotTo(HaveOccurred())
				zipReader, e := zip.NewReader(bytes.NewReader(responseBody), int64(len(responseBody)))
				Expect(e).NotTo(HaveOccurred())

				Expect(zipReader.File).To(HaveLen(1))
				VerifyZipFileEntry(zipReader, "somefile", "lalala\n\n")
			})
		})
	})

	Describe("/packages", func() {
		Describe("PUT", func() {
			Context("async=true", func() {
				It("returns StatusAccepted", func() {
					response, e := client.Do(
						newGetRequest("https://internal.127.0.0.1.nip.io:4443/sign/packages/myguid?verb=put", "the-username", "the-password"))
					Expect(e).NotTo(HaveOccurred())
					Expect(response.StatusCode).To(Equal(http.StatusOK))

					signedUrl, e := ioutil.ReadAll(response.Body)
					Expect(e).NotTo(HaveOccurred())

					r, e := httputil.NewPutRequest(string(signedUrl)+"&async=true", map[string]map[string]io.Reader{
						"package": map[string]io.Reader{"somefilename": CreateZip(map[string]string{"somefile": "lalala\n\n"})},
					})
					Expect(e).NotTo(HaveOccurred())
					response, e = client.Do(r)

					Expect(e).NotTo(HaveOccurred())
					Expect(response.StatusCode).To(Equal(http.StatusAccepted))
				})
			})
		})
	})
})

func newGetRequest(url string, username string, password string) *http.Request {
	request, e := http.NewRequest("GET", url, nil)
	Expect(e).NotTo(HaveOccurred())
	request.SetBasicAuth(username, password)
	return request
}

func GetStatusCode(response *http.Response) int { return response.StatusCode }
