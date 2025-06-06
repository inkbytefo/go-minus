# Vulkan "Hello Triangle" Örneği

Bu belge, GO-Minus programlama dili kullanarak Vulkan API ile basit bir üçgen çizme örneğini açıklamaktadır. Bu örnek, GO-Minus'un grafik programlama yeteneklerini göstermek için tasarlanmıştır.

## Gereksinimler

- GO-Minus 0.5.0 veya üzeri
- Vulkan SDK 1.3.0 veya üzeri
- GLFW 3.3 veya üzeri
- Desteklenen bir GPU ve sürücü

## Kurulum

### Vulkan SDK Kurulumu

1. [Vulkan SDK](https://vulkan.lunarg.com/sdk/home) adresinden işletim sisteminize uygun SDK'yı indirin ve kurun.
2. Ortam değişkenlerinin doğru şekilde ayarlandığından emin olun.

### GLFW Kurulumu

1. [GLFW](https://www.glfw.org/download.html) adresinden işletim sisteminize uygun GLFW'yi indirin ve kurun.
2. Kütüphane dosyalarının doğru şekilde ayarlandığından emin olun.

## Proje Yapısı

```
vulkan-hello-triangle/
├── main.gom
├── shaders/
│   ├── shader.vert
│   └── shader.frag
└── gom.mod
```

## Kod

### main.gom

```go
// main.gom
package main

import (
    "fmt"
    "os"
    "runtime"
    
    "vulkan"
    "glfw"
)

// Sabitler
const (
    WIDTH  = 800
    HEIGHT = 600
    TITLE  = "Vulkan Hello Triangle - GO-Minus"
)

// Vertex yapısı
struct Vertex {
    float[2] pos
    float[3] color
}

// Uygulama sınıfı
class VulkanApplication {
    private:
        // GLFW penceresi
        glfw.Window window
        
        // Vulkan nesneleri
        vulkan.Instance instance
        vulkan.PhysicalDevice physicalDevice
        vulkan.Device device
        vulkan.Queue graphicsQueue
        vulkan.Queue presentQueue
        vulkan.SwapchainKHR swapchain
        vulkan.Image[] swapchainImages
        vulkan.ImageView[] swapchainImageViews
        vulkan.RenderPass renderPass
        vulkan.PipelineLayout pipelineLayout
        vulkan.Pipeline graphicsPipeline
        vulkan.Framebuffer[] swapchainFramebuffers
        vulkan.CommandPool commandPool
        vulkan.CommandBuffer[] commandBuffers
        vulkan.Semaphore imageAvailableSemaphore
        vulkan.Semaphore renderFinishedSemaphore
        vulkan.Fence inFlightFence
        
        // Vertex buffer
        vulkan.Buffer vertexBuffer
        vulkan.DeviceMemory vertexBufferMemory
        
        // Pencere boyutları
        uint32 width
        uint32 height
        
        // Validation layers
        bool enableValidationLayers
        string[] validationLayers
        
    public:
        // Yapıcı metot
        VulkanApplication(uint32 width, uint32 height, string title) {
            this.width = width
            this.height = height
            
            // Validation layers
            this.enableValidationLayers = true
            this.validationLayers = []string{
                "VK_LAYER_KHRONOS_validation"
            }
            
            // GLFW başlatma
            if !glfw.Init() {
                throw new Exception("GLFW başlatılamadı")
            }
            
            // Vulkan desteği olmadan pencere oluşturma
            glfw.WindowHint(glfw.CLIENT_API, glfw.NO_API)
            glfw.WindowHint(glfw.RESIZABLE, glfw.FALSE)
            
            // Pencere oluşturma
            this.window = glfw.CreateWindow(width, height, title, null, null)
            if this.window == null {
                throw new Exception("GLFW penceresi oluşturulamadı")
            }
        }
        
        // Yıkıcı metot
        ~VulkanApplication() {
            // Vulkan kaynaklarını temizleme
            this.cleanup()
            
            // GLFW kaynaklarını temizleme
            glfw.DestroyWindow(this.window)
            glfw.Terminate()
        }
        
        // Uygulamayı çalıştırma
        void run() {
            this.initVulkan()
            this.mainLoop()
        }
        
        // Vulkan başlatma
        private void initVulkan() {
            this.createInstance()
            this.setupDebugMessenger()
            this.createSurface()
            this.pickPhysicalDevice()
            this.createLogicalDevice()
            this.createSwapchain()
            this.createImageViews()
            this.createRenderPass()
            this.createGraphicsPipeline()
            this.createFramebuffers()
            this.createCommandPool()
            this.createVertexBuffer()
            this.createCommandBuffers()
            this.createSyncObjects()
        }
        
        // Ana döngü
        private void mainLoop() {
            while !glfw.WindowShouldClose(this.window) {
                glfw.PollEvents()
                this.drawFrame()
            }
            
            // GPU işlemlerinin tamamlanmasını bekle
            this.device.waitIdle()
        }
        
        // Temizleme
        private void cleanup() {
            // Senkronizasyon nesnelerini temizleme
            this.device.destroySemaphore(this.imageAvailableSemaphore)
            this.device.destroySemaphore(this.renderFinishedSemaphore)
            this.device.destroyFence(this.inFlightFence)
            
            // Komut havuzunu temizleme
            this.device.destroyCommandPool(this.commandPool)
            
            // Vertex buffer'ı temizleme
            this.device.destroyBuffer(this.vertexBuffer)
            this.device.freeMemory(this.vertexBufferMemory)
            
            // Framebuffer'ları temizleme
            for framebuffer in this.swapchainFramebuffers {
                this.device.destroyFramebuffer(framebuffer)
            }
            
            // Pipeline'ı temizleme
            this.device.destroyPipeline(this.graphicsPipeline)
            this.device.destroyPipelineLayout(this.pipelineLayout)
            this.device.destroyRenderPass(this.renderPass)
            
            // Image view'ları temizleme
            for imageView in this.swapchainImageViews {
                this.device.destroyImageView(imageView)
            }
            
            // Swapchain'i temizleme
            this.device.destroySwapchainKHR(this.swapchain)
            
            // Mantıksal cihazı temizleme
            this.device.destroy()
            
            // Yüzeyi temizleme
            this.instance.destroySurfaceKHR(this.surface)
            
            // Debug messenger'ı temizleme
            if this.enableValidationLayers {
                this.instance.destroyDebugUtilsMessengerEXT(this.debugMessenger)
            }
            
            // Instance'ı temizleme
            this.instance.destroy()
        }
        
        // Vulkan instance oluşturma
        private void createInstance() {
            // Uygulama bilgisi
            appInfo := vulkan.ApplicationInfo{
                sType: vulkan.STRUCTURE_TYPE_APPLICATION_INFO,
                pApplicationName: "Hello Triangle",
                applicationVersion: vulkan.MAKE_VERSION(1, 0, 0),
                pEngineName: "No Engine",
                engineVersion: vulkan.MAKE_VERSION(1, 0, 0),
                apiVersion: vulkan.API_VERSION_1_0
            }
            
            // Instance oluşturma bilgisi
            createInfo := vulkan.InstanceCreateInfo{
                sType: vulkan.STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
                pApplicationInfo: &appInfo
            }
            
            // Gerekli uzantıları alma
            extensions := this.getRequiredExtensions()
            createInfo.enabledExtensionCount = uint32(len(extensions))
            createInfo.ppEnabledExtensionNames = extensions
            
            // Validation layers
            if this.enableValidationLayers {
                if !this.checkValidationLayerSupport() {
                    throw new Exception("Validation layers istendi ancak mevcut değil")
                }
                
                createInfo.enabledLayerCount = uint32(len(this.validationLayers))
                createInfo.ppEnabledLayerNames = this.validationLayers
                
                // Debug messenger oluşturma
                debugCreateInfo := this.populateDebugMessengerCreateInfo()
                createInfo.pNext = &debugCreateInfo
            } else {
                createInfo.enabledLayerCount = 0
                createInfo.pNext = null
            }
            
            // Instance oluşturma
            result := vulkan.CreateInstance(&createInfo, null, &this.instance)
            if result != vulkan.SUCCESS {
                throw new Exception("Vulkan instance oluşturulamadı")
            }
        }
        
        // Kare çizme
        private void drawFrame() {
            // Fence'i bekle
            this.device.waitForFences(1, [this.inFlightFence], true, uint64.MAX)
            this.device.resetFences(1, [this.inFlightFence])
            
            // Bir sonraki görüntüyü al
            imageIndex := uint32(0)
            result := this.device.acquireNextImageKHR(this.swapchain, uint64.MAX, this.imageAvailableSemaphore, null, &imageIndex)
            
            if result != vulkan.SUCCESS && result != vulkan.SUBOPTIMAL_KHR {
                throw new Exception("Swapchain görüntüsü alınamadı")
            }
            
            // Komut buffer'ı gönder
            submitInfo := vulkan.SubmitInfo{
                sType: vulkan.STRUCTURE_TYPE_SUBMIT_INFO
            }
            
            waitSemaphores := []vulkan.Semaphore{this.imageAvailableSemaphore}
            waitStages := []vulkan.PipelineStageFlags{vulkan.PIPELINE_STAGE_COLOR_ATTACHMENT_OUTPUT_BIT}
            submitInfo.waitSemaphoreCount = 1
            submitInfo.pWaitSemaphores = waitSemaphores
            submitInfo.pWaitDstStageMask = waitStages
            
            submitInfo.commandBufferCount = 1
            submitInfo.pCommandBuffers = [this.commandBuffers[imageIndex]]
            
            signalSemaphores := []vulkan.Semaphore{this.renderFinishedSemaphore}
            submitInfo.signalSemaphoreCount = 1
            submitInfo.pSignalSemaphores = signalSemaphores
            
            result = this.graphicsQueue.submit(1, [submitInfo], this.inFlightFence)
            if result != vulkan.SUCCESS {
                throw new Exception("Komut buffer gönderilemedi")
            }
            
            // Görüntüyü ekrana sun
            presentInfo := vulkan.PresentInfoKHR{
                sType: vulkan.STRUCTURE_TYPE_PRESENT_INFO_KHR
            }
            
            presentInfo.waitSemaphoreCount = 1
            presentInfo.pWaitSemaphores = signalSemaphores
            
            swapchains := []vulkan.SwapchainKHR{this.swapchain}
            presentInfo.swapchainCount = 1
            presentInfo.pSwapchains = swapchains
            presentInfo.pImageIndices = [imageIndex]
            
            result = this.presentQueue.presentKHR(&presentInfo)
            if result != vulkan.SUCCESS {
                throw new Exception("Görüntü sunulamadı")
            }
        }
}

// Ana fonksiyon
func main() {
    // Platform ayarları
    runtime.LockOSThread()
    
    try {
        // Uygulama oluşturma
        app := VulkanApplication(WIDTH, HEIGHT, TITLE)
        
        // Uygulamayı çalıştırma
        app.run()
    } catch (Exception e) {
        fmt.Println("Hata:", e.message)
        os.Exit(1)
    }
}
```

### shader.vert

```glsl
#version 450
#extension GL_ARB_separate_shader_objects : enable

layout(location = 0) in vec2 inPosition;
layout(location = 1) in vec3 inColor;

layout(location = 0) out vec3 fragColor;

void main() {
    gl_Position = vec4(inPosition, 0.0, 1.0);
    fragColor = inColor;
}
```

### shader.frag

```glsl
#version 450
#extension GL_ARB_separate_shader_objects : enable

layout(location = 0) in vec3 fragColor;

layout(location = 0) out vec4 outColor;

void main() {
    outColor = vec4(fragColor, 1.0);
}
```

### gom.mod

```
module vulkan-hello-triangle

go 1.18

require (
    vulkan v1.0.0
    glfw v3.3.0
)
```

## Derleme ve Çalıştırma

### Shader Derleme

Öncelikle, GLSL shader'ları SPIR-V formatına derlememiz gerekiyor:

```bash
# Vertex shader derleme
glslc shader.vert -o vert.spv

# Fragment shader derleme
glslc shader.frag -o frag.spv
```

### Uygulama Derleme ve Çalıştırma

```bash
# Derleme
gominus build

# Çalıştırma
./vulkan-hello-triangle
```

## Sonuç

Bu örnek, GO-Minus programlama dili kullanarak Vulkan API ile basit bir üçgen çizme işlemini göstermektedir. Örnek, Vulkan'ın temel kavramlarını (instance, device, swapchain, pipeline, vb.) ve GO-Minus'un nesne yönelimli programlama, istisna işleme ve bellek yönetimi özelliklerini göstermektedir.

## Kaynaklar

- [Vulkan Tutorial](https://vulkan-tutorial.com/)
- [Vulkan SDK](https://vulkan.lunarg.com/sdk/home)
- [GLFW](https://www.glfw.org/)
- [GO-Minus Belgelendirmesi](../../reference/README.md)
