
## Snapshotters

  

### Introduction

Snapshotters are one of the critical components of containerd's architecture; they are responsible for creating the container filesystem that will be used by containerd to unpack the image layers, essentially pulling the image. Containerd has a pluggable interface for the snapshotting module, which lets it support multiple snapshotters, even custom implementations, without the need for rebuilding the containerd image.


    
  ![snapshotter role in containerd](https://drive.google.com/uc?id=1nFqsasFLH0fhFYOVlndPw6SYBrdnCVp4)

### Built-in Snapshotters

Containerd comes with some built-in snapshotters by default, like Native, OverlayFS, BTRFS, etc.

  

#### OverlayFS Snapshotter

  

The OverlayFS snapshotter in containerd is one of the most widely used snapshotters. It relies on the Linux kernel’s OverlayFS, a union filesystem that layers directories on top of each other. This snapshotter creates layered mounts where each image layer becomes a read-only lower directory, with a writable upper directory created for the container writes.

It supports page cache sharing for multiple containers accessing the same file. This makes it memory efficient; however, it does copy up write operations when a new container is booted up. This can cause initial latency, but this would not be present for further container write operations.

  ![Overlayfs snapshotter architecture](https://drive.google.com/uc?id=1mN2fWbdzLHsURL96Mh7lhhl4gJ36i0rF)
[src](https://docs.docker.com/engine/storage/drivers/overlayfs-driver/)


#### Native Snapshotter

  

The native snapshotter is the simplest snapshotter, which creates full directory copies for each snapshot layer; it doesn't make use of any copy-on-write (COW) mechanisms like overlayfs or btrfs. It is storage inefficient as it duplicates the entire filesystem tree when creating a new snapshot.

  

#### BTRFS Snapshotter

  

BTRFS snapshotter makes use of BTRFS (B-tree filesystem) to manage container images and filesystems efficiently. It leverages the native snapshot and subvolume capabilities of the Btrfs filesystem. Snapshots in Btrfs can be lightweight copy-on-write references of an existing subvolume (read-only snapshot layer). It has fast container startup and minimal disc usage. It doesn't require layering mounts like overlayfs and offers built-in features like compression, checksumming, and deduplication.

  
  

### Plugin-based snapshots

  

#### Nydus Snapshotter

  

Nydus is a highly performant remote snapshotter; it essentially fetches data blocks from remote storage and registries on demand for containers. It uses a custom image format called "Rafs" where metadata and the actual image (blobs) are separated, which enables it to have features like chunk-level deduplication, reducing image size through ignoring whiteout files. It reduces cold-startup times of containers and also has features to support end-to-end security checks for images and users and compatibility with different storage backends.

  

It is currently being used on a large scale in Alibaba Cloud workloads and is also used in VM-based container runtimes like KataContainers.

  ![Nydus snapshotter architecture](https://drive.google.com/uc?id=1JIIJtDQKStP5XiM1-CvOgvDK7Vg2HxpT)
[src](https://docs.docker.com/engine/storage/drivers/overlayfs-driver/](https://d7y.io/blog/2022/01/17/containerd-accepted-nydus-snapshotter/))
  

#### Stargz Snapshotter

Stargz is also a remote snapshotter of containerd that enables "lazy pulling" of image layers, allowing containers to start before complete image download. It uses an eStargz container image format that is built on top of gzip and allows seeking through file contents, which are actually stored remotely, through its metadata (powered by an indexing). It makes use of the FUSE filesystem to mount eStargz layers directly from registries. This approach reduces container cold startup time significantly, especially when fewer files of an image are usually required at runtime.

  

Large-scale K8s cluster deployments typically use this snapshotter, allowing the reuse of heavy base images across multiple jobs.

  
  

### Performance & Trade‑offs

  

#### Overlay vs Other Native COW (BTRFS)

  

Native COW snapshotters like BTRFS have better copy-on-write functionalities when compared to overlayfs, as they make use of lightweight subvolumes or virtual blocks, unlike layered mounts like overlayfs. For the same reason, overlayfs has a higher startup latency, as it needs to perform a heavy copy-up operation for container inits and also has inefficient disc usage when compared to BTRFS and devmapper, which make use of virtual filesystems.

  

But in practice, OverlayFS is still the preferred inbuilt snapshotter, as Btrfs snapshotters face significant performance challenges with disc usage collection, as they perform expensive file system scans rather than utilising Btrfs's native quota features. This limitation can result in high CPU utilisation during regular containerd operations.

  

So the tradeoff involves balancing startup latency and disc usage against the overall performance of containerd with the snapshotter.

  

#### Plugin Tradeoffs

  

Remote snapshotters like Nydus and Stargz provide advanced features that provide super-efficient containerd performance for large-scale deployments, but they come with additional complexity. They do excel in environments with large images, high-volume deployments and during bandwidth limitations. However, they require additional overhead in terms of setup, like building the images into the custom format supported by these snapshotters and supporting the required components, like having the FUSE daemons and custom processes, storage backends, etc.

  

Hence, the tradeoff involves operational complexity against the performance benefits at large scale when choosing a remote snapshotter plugin for containerd.
