require 'rest-client'

module IntegrationHttp
  def endpoint
    'http://localhost:9292'
  end

  def make_get_request(path)
    RestClient.get "#{endpoint}#{path}"
  rescue => e
    e.response
  end

  def make_put_request(path, body)
    RestClient.put "#{endpoint}#{path}", body
  rescue => e
    e.response
  end
end
